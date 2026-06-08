from app.retrieval.hybrid_search import _profile_subject_boost


def test_profile_subject_boost_rewards_watched_subject_match():
    profile = {
        "SubjectPreferences": [
            {"SubjectName": "Fullstack Development"},
        ]
    }

    assert _profile_subject_boost("Fullstack Development", profile) == 0.08
    assert _profile_subject_boost("Cloud Computing", profile) == 0.0
