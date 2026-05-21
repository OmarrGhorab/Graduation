from app.tools.schemas import SearchRelevantCoursesInput, GetUserProfileInput


def test_search_relevant_courses_input_validates():
    item = SearchRelevantCoursesInput(user_id="123", query="backend", top_k=5, exclude_course_ids=["1"])
    assert item.user_id == "123"
    assert item.top_k == 5


def test_get_user_profile_input_requires_user_id():
    item = GetUserProfileInput(user_id="abc")
    assert item.user_id == "abc"
