import json
from typing import List, Dict

from app.utils.profile_utils import list_cart_subjects, list_subject_preferences, list_user_interests

def build_recommendation_prompt(user_profile: Dict, courses: List[Dict]) -> str:
    """
    Constructs a detailed prompt for Gemma 4 to generate personalized recommendations.
    Legacy v1 flow is intentionally kept for feature-flag fallback while v2 agentic path rolls out.
    """
    
    # Extract key analytics
    user_interests = list_user_interests(user_profile)
    cart_subjects = list_cart_subjects(user_profile)
    avg_completion = user_profile.get('AvgCompletionPct') or user_profile.get('avgCompletionPct') or 0.0
    engagement_score = user_profile.get('EngagementScore') or user_profile.get('engagementScore') or 0
    
    profile_text = f"""
Student Profile & Intent:
- Onboarding Interests: {', '.join(user_interests) if user_interests else 'General Learner'}
- High-Intent Subjects (In Cart): {', '.join(cart_subjects) if cart_subjects else 'None currently'}
- Avg Course Completion: {avg_completion:.1f}%
- Learning Behavior: {user_profile.get('CompletionTendency') or 'CONSISTENT'} finisher.

Engagement by Subject (Last 30 Days):
"""
    subject_preferences = list_subject_preferences(user_profile)
    for stat in subject_preferences:
        subject_name = stat.get('SubjectName') or stat.get('subjectName') or 'Unknown'
        watch_seconds = stat.get('TotalWatchTime', stat.get('totalWatchTime', 0)) or 0
        watch_minutes = int(watch_seconds / 60)
        engagement = stat.get('AvgEngagement', stat.get('avgEngagement', 0))
        profile_text += f"- {subject_name}: {watch_minutes} mins (Avg Engagement: {engagement}%)\n"
    
    # 3. Build Course Catalog for AI
    catalog_text = "AVAILABLE COURSES CATALOG:\n"
    for course in (courses or [])[:20]: 
        rating = course.get('Rating') or 4.5
        enrollment = course.get('enrollmentCount') or 0
        catalog_text += f"- ID: {course.get('id')} | {course.get('title')}\n"
        subject_data = course.get('subject', {})
        subject_name = subject_data.get('name') if isinstance(subject_data, dict) else None
        subject_name = subject_name or course.get('subjectName') or 'N/A'
        catalog_text += f"  Subject: {subject_name} | Enrolled Students: {enrollment}\n"
        catalog_text += f"  Teacher Rating: {rating}/5.0 | Desc: {course.get('description', '')[:80]}...\n\n"

    # 4. Final System Instructions
    system_instruction = f"""
Act as a precise Educational Advisor.
Goal: Predict the next 6 courses this student will likely purchase or engage with most.

PRIORITIZATION HIERARCHY:
1. HIGH INTENT: Subjects currently in the student's CART or declared as INTERESTS.
2. ENGAGEMENT: Subjects where the student has high watch time and >50% completion.
3. TEACHER AUTHORITY: Prefer courses with high 'Enrolled Students' counts (indicates social proof/quality).
4. VARIETY: If they are 100% done with a subject, suggest the 'Advanced' version or a related 'Cloud' infrastructure version.

OUTPUT: Return ONLY a JSON list of objects: [{{"courseId": "...", "score": 0-100, "matchReason": "...", "priority": "HIGH/MEDIUM/LOW"}}]
"""

    prompt = f"""
    {system_instruction}

    ### DATA INPUTS
    {profile_text}
    - Overall System Engagement Score: {engagement_score}/100
    - Preferred Device: {user_profile.get('preferredDevice') or 'MOBILE'}

    ### CATALOG FOR SELECTION
    {catalog_text}

    ### INSTRUCTION
    Analyze the overlap between intent (Cart/Interests) and historical success (Engagement). 
    Weight Cart subjects at 1.5x. Weight Teacher popularity at 1.2x.
    
    Return top 6 recommendations in JSON format.
    """
    
    return prompt
