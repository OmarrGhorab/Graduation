import json
from typing import List, Dict

def build_recommendation_prompt(user_profile: Dict, courses: List[Dict]) -> str:
    """
    Constructs a detailed prompt for Gemma 4 to generate personalized recommendations.
    """
    
    # Extract key analytics
    user_interests = user_profile.get('UserInterests') or []
    avg_completion = user_profile.get('AvgCompletionPct') or 0.0
    engagement_score = user_profile.get('EngagementScore') or 0
    avg_session = user_profile.get('averageSessionDuration') or 0
    
    profile_text = f"""
Student Engagement Profile:
- Avg Completion Rate: {avg_completion:.1f}%
- Completion Tendency: {user_profile.get('CompletionTendency') or 'MEDIUM'}
- Explicit Interests (Settings): {', '.join(user_interests)}
- Known Strengths (High Grades): {', '.join(['Mathematics (Excellence)', 'Physics (Strong)'])} 

Historical Interests (base on watch time):
"""
    subject_preferences = user_profile.get('SubjectPreferences') or []
    for stat in subject_preferences:
        profile_text += f"- {stat.get('SubjectName', 'Unknown')}: {stat.get('TotalWatchMinutes', 0)} mins\n"
    
    # 3. Build Course Catalog for AI
    catalog_text = "AVAILABLE COURSES:\n"
    for course in (courses or [])[:15]: 
        rating = course.get('Rating') or 4.5
        catalog_text += f"- ID: {course.get('id')} | {course.get('title')} (Rating: {rating}/5.0)\n"
        catalog_text += f"  Subject: {course.get('subject', {}).get('name', 'N/A')} | Teacher: {course.get('TeacherName', 'Senior Instuctor')}\n"
        catalog_text += f"  Desc: {course.get('description', '')[:80]}...\n\n"

    # 4. Final System Instructions (Optimized for speed)
    system_instruction = f"""
Act as a personal career advisor for an E-Learning platform.
Goal: Select exactly 6 courses the student will LOVE based on their strength regions and teacher quality.

ADVICE RULES:
1. Prioritize courses with High Ratings (4.7+).
2. If the user has a "Strength", suggest a "Next Step" course.
3. Keep descriptions punchy and exciting.
4. ONLY return a JSON list of objects: [{{"courseId": "...", "score": 85, "matchReason": "...", "priority": "HIGH"}}]
"""

    prompt = f"""
    {system_instruction}

    ### STUDENT PROFILE
    {profile_text}
    - Overall Engagement Score: {engagement_score}/100
    - Behavior: {avg_session} min avg session on {user_profile.get('preferredDevice') or 'any'} devices.

    ### AVAILABLE COURSES CATALOG
    {catalog_text}

    ### TASK
    Rate each available course based on how well it fits the student's profile.
    
    ### SCORING CRITERIA
    - Subject Affinity: Matches top watched subjects.
    - Engagement: High-quality matches for high-score students.

    ### OUTPUT FORMAT
    Return ONLY a JSON array of recommendation objects.
    
    Example format:
    [
      {{"courseId": "...", "score": 85, "matchReason": "...", "priority": "HIGH"}}
    ]

    Only include the top 6 courses with a score > 40.
    """
    
    return prompt
