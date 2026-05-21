from app.clustering.feature_builder import build_user_behavior_summary, build_user_numeric_features


def test_build_user_behavior_summary_contains_key_signals():
    summary = build_user_behavior_summary("u1", {
        "UserInterests": ["Math"],
        "CartSubjects": ["Science"],
        "CompletionTendency": "HIGH",
        "AvgCompletionPct": 82,
        "AvgSessionDuration": 120,
        "SubjectPreferences": [{"SubjectName": "Backend"}],
        "PreviewInterests": [{"SubjectName": "Databases"}],
    })
    assert "Math" in summary
    assert "Backend" in summary
    assert "Databases" in summary


def test_build_user_numeric_features_handles_analytics():
    features = build_user_numeric_features({
        "AllAnalytics": [
            {"CourseID": "1", "TotalWatchTime": 100, "CompletionPct": 50, "EngagementScore": 70, "SubjectName": "CS"},
            {"CourseID": "1", "TotalWatchTime": 50, "CompletionPct": 60, "EngagementScore": 80, "SubjectName": "CS"},
        ],
        "CartSubjects": ["Science"],
    })
    assert features["courses_count"] == 1.0
    assert features["watch_time_total"] == 150.0
    assert features["cart_subjects_count"] == 1.0
