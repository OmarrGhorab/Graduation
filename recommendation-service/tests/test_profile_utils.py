from app.utils.profile_utils import build_behavior_query, get_enrolled_ids_from_profile


def test_build_behavior_query_prefers_watch_time_subjects():
    query = build_behavior_query(
        {
            "SubjectPreferences": [
                {"SubjectName": "Data Science", "TotalWatchTime": 4590},
            ],
            "UserInterests": ["Cloud Architecture"],
        }
    )

    assert "watched Data Science" in query
    assert "Cloud Architecture" in query


def test_profile_helpers_accept_public_camel_case_payloads():
    profile = {
        "courseAnalytics": [{"courseId": "course-1"}],
        "subjectPreferences": [{"subjectName": "Machine Learning"}],
        "watchPatterns": {"completionTendency": "HIGH"},
    }

    assert get_enrolled_ids_from_profile(profile) == ["course-1"]
    assert "watched Machine Learning" in build_behavior_query(profile)
    assert "HIGH completion learner" in build_behavior_query(profile)
