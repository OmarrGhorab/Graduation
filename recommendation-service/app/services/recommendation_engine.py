from app.services.course_client import course_client
from app.services.gemma_client import gemma_client
from app.utils.prompt_builder import build_recommendation_prompt
from app.config import settings
from app.agents.recommendation_agent import recommendation_agent
from app.models.database import SessionLocal
from app.models.recommendation import RecommendationHistory
import logging
import json
import redis.asyncio as redis
import uuid
from opentelemetry import trace

logger = logging.getLogger(__name__)
tracer = trace.get_tracer(__name__)

# Initialize Redis
redis_conn = redis.from_url(settings.REDIS_URL, decode_responses=True)

async def get_personalized_recommendations(user_id: str):
    if settings.AGENT_RECOMMENDATIONS_ENABLED:
        return await _get_agentic_recommendations(user_id)
    return await _get_legacy_recommendations(user_id)


async def _get_agentic_recommendations(user_id: str):
    cache_key = f"recommendation:v2:{user_id}"
    explain_key = f"recommendation:v2:explain:{user_id}"

    try:
        cached_data = await redis_conn.get(cache_key)
        if cached_data:
            logger.info(f"V2 cache hit for user {user_id}")
            return json.loads(cached_data)
    except Exception as e:
        logger.warning(f"Redis error: {str(e)}")

    result = await recommendation_agent.recommend(user_id)
    recommendations = result.get("recommendations", [])
    reasoning_summary = result.get("reasoning_summary", "")
    tool_trace = result.get("tool_trace", [])
    errors = result.get("errors", [])

    for rec in recommendations:
        rec.setdefault("source", ["semantic_similarity"])
        rec.setdefault("clusterContribution", 0.0)
        rec.setdefault("confidence", float(rec.get("score", 0)) / 100.0)
        rec.setdefault("reason", rec.get("matchReason", "Recommended by agentic retrieval and ranking."))

    try:
        await redis_conn.setex(
            cache_key,
            settings.RECOMMENDATION_CACHE_TTL,
            json.dumps(recommendations)
        )
        await redis_conn.setex(
            explain_key,
            settings.AGENT_REASONING_LOG_TTL,
            json.dumps(
                {
                    "reasoningSummary": reasoning_summary,
                    "toolTrace": tool_trace,
                    "errors": errors,
                }
            ),
        )
        logger.info(
            "recommendation_reasoning_trace_cached",
            extra={
                "user_id": str(user_id),
                "tool_calls": len(tool_trace),
                "error_count": len(errors),
                "recommendation_count": len(recommendations),
            },
        )
    except Exception as e:
        logger.warning(f"Failed to cache v2 recommendations/explanation: {str(e)}")

    _persist_v2_recommendation_history(user_id, recommendations)
    return recommendations


async def _get_legacy_recommendations(user_id: str):
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


def _persist_v2_recommendation_history(user_id: str, recommendations: list):
    db = SessionLocal()
    try:
        try:
            user_uuid = uuid.UUID(str(user_id))
        except Exception:
            user_uuid = None

        for rec in recommendations:
            try:
                course_id_raw = rec.get("courseId")
                course_uuid = uuid.UUID(str(course_id_raw)) if course_id_raw else None
            except Exception:
                course_uuid = None

            row = RecommendationHistory(
                user_id=user_uuid,
                course_id=course_uuid,
                score=float(rec.get("score", 0)),
                match_reason=rec.get("reason") or rec.get("matchReason"),
                match_type="agentic_v2",
                model_version=f"{settings.AI_MODEL}:agentic-v2",
            )
            db.add(row)
        db.commit()
    except Exception as exc:
        db.rollback()
        logger.warning(f"Failed to persist v2 recommendation history: {exc}")
    finally:
        db.close()


async def get_recommendation_explanation(user_id: str):
    explain_key = f"recommendation:v2:explain:{user_id}"
    with tracer.start_as_current_span("recommendation.reasoning.trace.read") as span:
        span.set_attribute("recommendation.user_id", str(user_id))
        data = await redis_conn.get(explain_key)
        if not data:
            logger.info("recommendation_reasoning_trace_missing", extra={"user_id": str(user_id)})
            span.set_attribute("recommendation.trace_found", False)
            return {
                "reasoningSummary": "",
                "toolTrace": [],
                "errors": [],
            }
        try:
            parsed = json.loads(data)
            span.set_attribute("recommendation.trace_found", True)
            span.set_attribute("recommendation.tool_calls", len(parsed.get("toolTrace", [])))
            return parsed
        except Exception:
            logger.warning("recommendation_reasoning_trace_parse_failed", extra={"user_id": str(user_id)})
            span.set_attribute("recommendation.trace_found", True)
            span.set_attribute("recommendation.trace_parse_failed", True)
            return {
                "reasoningSummary": "",
                "toolTrace": [],
                "errors": ["failed_to_parse_explanation"],
            }

async def clear_cache(user_id: str):
    """Removes the cached recommendations for a specific user."""
    await redis_conn.delete(f"recommendation:v1:{user_id}")
    await redis_conn.delete(f"recommendation:v2:{user_id}")
    await redis_conn.delete(f"recommendation:v2:explain:{user_id}")
    # Retrieval results bake in the hybrid (incl. cluster) score, so drop them too
    # — otherwise a fresh agent run replays stale candidates without the boost.
    try:
        keys = await redis_conn.keys("retrieval:v1:*")
        if keys:
            await redis_conn.delete(*keys)
    except Exception as exc:
        logger.warning(f"Failed to flush retrieval cache: {exc}")
    logger.info(f"Cache cleared for user: {user_id}")

async def get_trending_recommendations():
    """Ranks courses globally by enrollment count and teacher authority with Redis caching."""
    cache_key = "recommendations:v1:trending"
    
    # 1. Check Cache
    try:
        cached_data = await redis_conn.get(cache_key)
        if cached_data:
            logger.info("Cache hit for trending recommendations")
            return json.loads(cached_data)
    except Exception as e:
        logger.warning(f"Redis error fetching trending cache: {str(e)}")

    # 2. Cache Miss - Fetch and Calculate
    logger.info("Cache miss for trending. Re-calculating...")
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
    
    # 3. Cache Results (1 hour = 3600 seconds)
    try:
        await redis_conn.setex(
            cache_key,
            settings.TRENDING_CACHE_TTL,
            json.dumps(result)
        )
        logger.info("Trending recommendations cached for 1 hour")
    except Exception as e:
        logger.warning(f"Failed to cache trending recommendations: {str(e)}")

    return result
