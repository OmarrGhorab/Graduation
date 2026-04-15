from app.services.course_client import course_client
from app.services.gemma_client import gemma_client
from app.utils.prompt_builder import build_recommendation_prompt
from app.config import settings
import logging
import json
import redis.asyncio as redis

logger = logging.getLogger(__name__)

# Initialize Redis
redis_conn = redis.from_url(settings.REDIS_URL, decode_responses=True)

async def get_personalized_recommendations(user_id: str):
    """
    Orchestrates the full recommendation flow:
    1. Check cache
    2. Fetch user analytics + course catalog
    3. Generate AI prompt
    4. Call Gemma 4
    5. Cache and return results
    """
    
    # 1. Check Cache
    cache_key = f"recommendation:v1:{user_id}"
    try:
        cached_data = await redis_conn.get(cache_key)
        if cached_data:
            logger.info(f"Cache hit for user {user_id}")
            return json.loads(cached_data)
    except Exception as e:
        logger.warning(f"Redis error: {str(e)}")

    # 2. Fetch Data
    logger.info(f"Cache miss for user {user_id}. Fetching fresh data...")
    try:
        user_profile = await course_client.get_user_analytics_profile(user_id)
        logger.info(f"Fetched user profile: {user_profile is not None}")
        
        all_courses = await course_client.get_all_courses()
        logger.info(f"Fetched {len(all_courses) if all_courses else 0} courses")
        
        if not all_courses:
            logger.warning("No courses available for recommendation")
            return []

        # Filter out already enrolled courses
        enrolled_course_ids = set()
        if user_profile and "AllAnalytics" in user_profile:
            enrolled_course_ids = {a.get("CourseID") for a in user_profile["AllAnalytics"]}
            logger.info(f"User is already enrolled in {len(enrolled_course_ids)} courses. Filtering...")
        
        filtered_courses = [c for c in all_courses if c['id'] not in enrolled_course_ids]
        logger.info(f"Remaining courses for AI to consider: {len(filtered_courses)}")
        
        if not filtered_courses:
            logger.warning("All available courses are already enrolled by the user.")
            return []

        # 3. Build Prompt
        logger.info("Building AI prompt...")
        prompt = build_recommendation_prompt(user_profile, filtered_courses)
        logger.info("Prompt built successfully")
        
        # 4. Generate with Gemma 4
        logger.info(f"Calling AI model: {settings.AI_MODEL}")
        recommendations = await gemma_client.generate_recommendations(prompt)
        logger.info(f"AI returned {len(recommendations) if isinstance(recommendations, list) else 'invalid'} items")
        
        # 5. Hydrate Results with full course data
        logger.info("Hydrating AI results with course catalog details...")
        hydrated_results = []
        
        # Create a lookup map for faster processing
        course_map = {c['id']: c for c in all_courses}
        
        if isinstance(recommendations, list):
            for rec in recommendations:
                course_id = rec.get("courseId")
                course_data = course_map.get(course_id)
                
                if course_data:
                    # Enrich with real DB data
                    rec["title"] = course_data.get("title")
                    rec["courseImage"] = course_data.get("courseImage") or "https://via.placeholder.com/300x160?text=Course+Image"
                    rec["price"] = course_data.get("price", 0)
                    rec["currency"] = course_data.get("currency", "EGP")
                    rec["enrolledCount"] = course_data.get("enrollmentCount", 0)
                    rec["subjectName"] = course_data.get("subjectName")
                    
                    # Real Teacher Details from Go Internal API
                    rec["teacher"] = {
                        "name": course_data.get("teacherName", "Unknown Instructor"),
                        "avatar": course_data.get("teacherProfileImg") or "https://i.pravatar.cc/150?u=fallback"
                    }
                    
                    hydrated_results.append(rec)
            
            recommendations = hydrated_results
            logger.info(f"Final hydrated response contains {len(recommendations)} valid courses.")

        # 6. Cache Results
        try:
            if isinstance(recommendations, list):
                await redis_conn.setex(
                    cache_key,
                    settings.RECOMMENDATION_CACHE_TTL,
                    json.dumps(recommendations)
                )
        except Exception as e:
            logger.warning(f"Failed to cache recommendations: {str(e)}")

        return recommendations
    except Exception as e:
        logger.error(f"CRITICAL: recommendation_engine failure: {str(e)}", exc_info=True)
        raise

async def clear_cache(user_id: str):
    """Removes the cached recommendations for a specific user."""
    cache_key = f"recommendation:v1:{user_id}"
    await redis_conn.delete(cache_key)
    logger.info(f"Cache cleared for user: {user_id}")

async def get_trending_recommendations():
    """Ranks courses globally by enrollment count and teacher authority."""
    all_courses = await course_client.get_all_courses()
    if not all_courses:
        return []
    
    # Sort by Enrollment Count (Primary) and Teacher Authority (Secondary)
    # We use a simple scoring: (Enrollments * 1.5) + (TeacherAuthority * 0.5)
    def calculate_trending_score(c):
        enrollments = c.get('enrollmentCount', 0)
        authority = c.get('teacherAuthority', 0)
        return (enrollments * 1.5) + (authority * 0.5)

    # Attach score for visibility if needed
    for c in all_courses:
        c['trendingScore'] = calculate_trending_score(c)

    # Sort descending
    sorted_courses = sorted(all_courses, key=calculate_trending_score, reverse=True)
    
    # Take Top 10
    trending = sorted_courses[:10]
    
    # Map to frontend-friendly format
    result = []
    for c in trending:
        result.append({
            "courseId": c.get('id'),
            "score": int(c.get('trendingScore', 0)),
            "title": c.get('title'),
            "courseImage": c.get('courseImage') or "https://via.placeholder.com/300x160?text=Trending",
            "price": c.get('price'),
            "currency": c.get('currency'),
            "enrolledCount": c.get('enrollmentCount', 0),
            "subjectName": c.get('subjectName'),
            "teacher": {
                "name": c.get('teacherName', "Prof. Omar Ghorab"),
                "avatar": c.get('teacherProfileImg') or "https://i.pravatar.cc/150?u=fallback"
            },
            "matchReason": "Currently trending among all students.",
            "priority": "HIGH"
        })
    
    return result
