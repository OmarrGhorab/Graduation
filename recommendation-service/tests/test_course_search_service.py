import pytest
from types import SimpleNamespace

from app.services import course_search_service


COURSES = [
    {
        "id": "course-python-1",
        "title": "Python for Data Analysis",
        "description": "Learn pandas, numpy, and analysis workflows.",
        "subjectId": "sub-data",
        "subjectName": "Data Science",
        "teacherId": "teacher-1",
        "teacherName": "Dr. Data",
        "teacherProfileImg": "teacher1.png",
        "teacherRating": 4.8,
        "courseImage": "python.png",
        "courseRating": 4.7,
        "totalRatings": 120,
        "enrolledStudents": 430,
        "deliveryType": "ONLINE",
        "locationName": "",
        "locationLat": None,
        "locationLng": None,
        "geofenceRadiusM": 0,
        "totalLessons": 20,
        "attendanceWindowMinutes": 15,
        "price": 50,
        "currency": "USD",
        "isPaid": True,
        "billingType": "ONE_TIME",
        "status": "ACTIVE",
        "attendanceWeight": 0.2,
        "previewVideoUrl": "",
        "previewVideoPublicId": "",
        "reminderIntervals": "15,10,5",
        "createdAt": "2026-01-01T00:00:00Z",
        "updatedAt": "2026-01-01T00:00:00Z",
    },
    {
        "id": "course-web-1",
        "title": "Modern Web Apps",
        "description": "Frontend architecture with React and API integration.",
        "subjectId": "sub-web",
        "subjectName": "Web Development",
        "teacherId": "teacher-2",
        "teacherName": "Ms. Web",
        "teacherProfileImg": "teacher2.png",
        "teacherRating": 4.6,
        "courseImage": "web.png",
        "courseRating": 4.5,
        "totalRatings": 88,
        "enrolledStudents": 510,
        "deliveryType": "ONLINE",
        "locationName": "",
        "locationLat": None,
        "locationLng": None,
        "geofenceRadiusM": 0,
        "totalLessons": 18,
        "attendanceWindowMinutes": 15,
        "price": 40,
        "currency": "USD",
        "isPaid": True,
        "billingType": "ONE_TIME",
        "status": "ACTIVE",
        "attendanceWeight": 0.2,
        "previewVideoUrl": "",
        "previewVideoPublicId": "",
        "reminderIntervals": "15,10,5",
        "createdAt": "2026-01-01T00:00:00Z",
        "updatedAt": "2026-01-01T00:00:00Z",
    },
    {
        "id": "course-cloud-1",
        "title": "Cloud Foundations",
        "description": "Deploy services and scale backend systems.",
        "subjectId": "sub-cloud",
        "subjectName": "Cloud Computing",
        "teacherId": "teacher-3",
        "teacherName": "Mr. Cloud",
        "teacherProfileImg": "teacher3.png",
        "teacherRating": 4.2,
        "courseImage": "cloud.png",
        "courseRating": 4.1,
        "totalRatings": 54,
        "enrolledStudents": 150,
        "deliveryType": "OFFLINE",
        "locationName": "Lab A",
        "locationLat": None,
        "locationLng": None,
        "geofenceRadiusM": 0,
        "totalLessons": 12,
        "attendanceWindowMinutes": 10,
        "price": 30,
        "currency": "USD",
        "isPaid": False,
        "billingType": "ONE_TIME",
        "status": "ACTIVE",
        "attendanceWeight": 0.2,
        "previewVideoUrl": "",
        "previewVideoPublicId": "",
        "reminderIntervals": "15,10,5",
        "createdAt": "2026-01-01T00:00:00Z",
        "updatedAt": "2026-01-01T00:00:00Z",
    },
]

SCIENCE_COURSES = [
    {
        **COURSES[0],
        "id": "course-chem-1",
        "title": "Secondary Chemistry Exam Prep",
        "description": "Chemistry reactions, acids, bases, and lab problem solving.",
        "subjectId": "sub-chem",
        "subjectName": "Chemistry",
    },
    {
        **COURSES[1],
        "id": "course-physics-1",
        "title": "Primary Physics Discovery Lab",
        "description": "Physics motion, energy, force, and simple experiments for school students.",
        "subjectId": "sub-physics",
        "subjectName": "Physics",
    },
]

ALIAS_COURSES = [
    {
        **COURSES[0],
        "id": "course-ml-1",
        "title": "Machine Learning Foundations",
        "description": "Build and evaluate machine learning models.",
        "subjectId": "sub-ml",
        "subjectName": "AI & Machine Learning",
    },
    {
        **COURSES[1],
        "id": "course-js-1",
        "title": "JavaScript for Web Apps",
        "description": "Modern JavaScript for frontend and backend web apps.",
        "subjectId": "sub-js",
        "subjectName": "Fullstack Development",
    },
]


def test_catalog_term_correction_handles_transposed_subject_typo():
    catalog_terms = {"chemistry", "physics", "biology"}

    assert course_search_service._best_catalog_term("phsyi", catalog_terms) == "physics"
    assert course_search_service._best_catalog_term("phyiscis", catalog_terms) == "physics"


def test_catalog_phrase_aliases_are_built_from_live_course_phrases():
    catalog_terms = course_search_service._collect_catalog_terms(ALIAS_COURSES)
    aliases = course_search_service._catalog_aliases(ALIAS_COURSES, catalog_terms)

    assert aliases["ml"] == "machine learning"
    assert any("javascript" in value for value in aliases.values())


class FakeVectorStore:
    def __init__(self, hits):
        self.hits = hits

    async def ensure_collections(self, _dim):
        return None

    async def search_courses(self, _query_vector, top_k=None, filters=None):
        return self.hits[: top_k or len(self.hits)]


@pytest.fixture(autouse=True)
def clear_fake_cache():
    fake_cache = {}

    async def fake_get(key):
        return fake_cache.get(key)

    async def fake_setex(key, ttl, value):
        fake_cache[key] = value

    course_search_service.redis_conn.get = fake_get
    course_search_service.redis_conn.setex = fake_setex


@pytest.mark.asyncio
async def test_semantic_search_returns_relevant_course_for_fuzzy_query(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-python-1", "similarityScore": 0.93, "payload": {}},
                {"courseId": "course-web-1", "similarityScore": 0.44, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "pythn analy", {}, page=1, limit=10)

    assert result["meta"]["total"] == 2
    assert result["data"][0]["id"] == "course-python-1"
    assert result["data"][0]["matchSource"] == "semantic"
    assert result["data"][0]["similarityScore"] == 0.93


@pytest.mark.asyncio
async def test_semantic_search_applies_filters_after_hydration(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-cloud-1", "similarityScore": 0.88, "payload": {}},
                {"courseId": "course-python-1", "similarityScore": 0.86, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search(
        "user-1",
        "cloud",
        {"deliveryType": "ONLINE", "isPaid": True},
        page=1,
        limit=10,
    )

    assert [item["id"] for item in result["data"]] == ["course-python-1"]


@pytest.mark.asyncio
async def test_personalization_only_nudges_close_matches(monkeypatch):
    profile = {
        "subjectPreferences": [{"subjectName": "Web Development"}],
    }
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value(profile))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-python-1", "similarityScore": 0.82, "payload": {}},
                {"courseId": "course-web-1", "similarityScore": 0.80, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {"course-web-1": 1.0})

    result = await course_search_service.semantic_course_search("user-1", "python basics", {}, page=1, limit=10)

    assert result["data"][0]["id"] == "course-python-1"
    assert result["data"][1]["id"] == "course-web-1"


@pytest.mark.asyncio
async def test_autocomplete_returns_course_and_subject_suggestions(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(COURSES))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-python-1", "similarityScore": 0.92, "payload": {}},
                {"courseId": "course-web-1", "similarityScore": 0.80, "payload": {}},
            ]
        ),
    )

    result = await course_search_service.course_autocomplete("user-1", "py", limit=8)

    assert any(item["type"] == "course" for item in result)
    assert any(item["type"] == "subject" for item in result)


@pytest.mark.asyncio
async def test_autocomplete_prefers_lexical_python_match_for_short_query(monkeypatch):
    lexical_courses = [
        COURSES[0],
        {
            **COURSES[1],
            "id": "course-de-1",
            "title": "Complete Spark Fundamentals Workshop",
            "subjectName": "Data Engineering",
        },
    ]
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(lexical_courses))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-de-1", "similarityScore": 0.95, "payload": {}},
                {"courseId": "course-python-1", "similarityScore": 0.61, "payload": {}},
            ]
        ),
    )

    result = await course_search_service.course_autocomplete("user-1", "pyth", limit=8)

    course_results = [item for item in result if item["type"] == "course"]
    assert course_results[0]["courseId"] == "course-python-1"


@pytest.mark.asyncio
async def test_autocomplete_short_query_filters_irrelevant_semantic_neighbors(monkeypatch):
    non_python = [
        {
            **COURSES[1],
            "id": "course-de-2",
            "title": "Practical Streaming Data Bootcamp",
            "subjectName": "Data Engineering",
        }
    ]
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(non_python))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-de-2", "similarityScore": 0.98, "payload": {}},
            ]
        ),
    )

    result = await course_search_service.course_autocomplete("user-1", "pyth", limit=8)

    assert result == []


@pytest.mark.asyncio
async def test_fallback_returns_keyword_matches_when_semantic_empty(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(course_search_service, "vector_store", FakeVectorStore([]))
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "react", {}, page=1, limit=10)

    assert result["meta"]["total"] == 1
    assert result["data"][0]["id"] == "course-web-1"
    assert result["data"][0]["matchSource"].startswith("lexical:")


@pytest.mark.asyncio
async def test_recent_searches_boost_related_course(monkeypatch):
    fake_cache = {}

    async def fake_get(key):
        return fake_cache.get(key)

    async def fake_setex(key, ttl, value):
        fake_cache[key] = value

    course_search_service.redis_conn.get = fake_get
    course_search_service.redis_conn.setex = fake_setex

    await course_search_service.log_recent_search("user-1", "python")

    lexical_courses = [
        {
            **COURSES[0],
            "id": "course-python-2",
            "title": "Python Automation Essentials",
        },
        {
            **COURSES[1],
            "id": "course-web-2",
            "title": "Modern Web Apps",
        },
    ]

    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(lexical_courses))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-web-2", "similarityScore": 0.62, "payload": {}},
                {"courseId": "course-python-2", "similarityScore": 0.55, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "automation", {}, page=1, limit=10)

    assert result["data"][0]["id"] == "course-python-2"


@pytest.mark.asyncio
async def test_search_diversifies_top_results_by_subject(monkeypatch):
    diversified_courses = [
        {**COURSES[0], "id": "course-python-a", "title": "Python A", "subjectName": "Data Science"},
        {**COURSES[0], "id": "course-python-b", "title": "Python B", "subjectName": "Data Science"},
        {**COURSES[0], "id": "course-python-c", "title": "Python C", "subjectName": "Data Science"},
        {**COURSES[1], "id": "course-web-a", "title": "Web A", "subjectName": "Web Development"},
    ]
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(diversified_courses))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-python-a", "similarityScore": 0.91, "payload": {}},
                {"courseId": "course-python-b", "similarityScore": 0.90, "payload": {}},
                {"courseId": "course-python-c", "similarityScore": 0.89, "payload": {}},
                {"courseId": "course-web-a", "similarityScore": 0.75, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "python", {}, page=1, limit=3)

    subjects = [item["subjectName"] for item in result["data"]]
    assert subjects.count("Data Science") <= 2
    assert "Web Development" in subjects


@pytest.mark.asyncio
async def test_query_expansion_helps_hands_on_match_practical_course(monkeypatch):
    expanded_courses = [
        {
            **COURSES[1],
            "id": "course-practical-1",
            "title": "Practical Backend Workshop",
            "description": "A project based applied lab for backend systems.",
            "subjectName": "Web Development",
        }
    ]
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(expanded_courses))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda text, **_kwargs: _async_value([0.1, 0.2, 0.3] if "practical" in text else [0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore([]),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "hands-on", {}, page=1, limit=10)

    assert result["meta"]["total"] == 1
    assert result["data"][0]["id"] == "course-practical-1"


@pytest.mark.asyncio
async def test_search_feedback_boosts_clicked_course(monkeypatch):
    fake_cache = {}

    async def fake_get(key):
        return fake_cache.get(key)

    async def fake_setex(key, ttl, value):
        fake_cache[key] = value

    course_search_service.redis_conn.get = fake_get
    course_search_service.redis_conn.setex = fake_setex

    await course_search_service.record_search_feedback("python", "course-python-1", "click")
    await course_search_service.record_search_feedback("python", "course-python-1", "enroll")

    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-web-1", "similarityScore": 0.73, "payload": {}},
                {"courseId": "course-python-1", "similarityScore": 0.69, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "python", {}, page=1, limit=10)

    assert result["data"][0]["id"] == "course-python-1"


@pytest.mark.asyncio
async def test_gibberish_query_returns_zero_results_when_semantic_only_matches_are_weak(monkeypatch):
    gibberish_courses = [
        {**COURSES[1], "id": "course-ui-1", "title": "Deep Dive Design Systems Workshop", "subjectName": "UI/UX Design"},
        {**COURSES[0], "id": "course-ds-1", "title": "Deep Dive Applied Regression Handbook", "subjectName": "Data Science"},
        {**COURSES[2], "id": "course-sec-1", "title": "Advanced Security Monitoring Bootcamp", "subjectName": "Cybersecurity"},
    ]
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(gibberish_courses))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-ui-1", "similarityScore": 0.55, "payload": {}},
                {"courseId": "course-ds-1", "similarityScore": 0.54, "payload": {}},
                {"courseId": "course-sec-1", "similarityScore": 0.53, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "fsaf", {}, page=1, limit=10)

    assert result["meta"]["total"] == 0
    assert result["data"] == []


@pytest.mark.asyncio
async def test_fuzzy_real_query_still_returns_results_despite_no_exact_spelling(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-python-1", "similarityScore": 0.88, "payload": {}},
                {"courseId": "course-web-1", "similarityScore": 0.57, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "pythn analy", {}, page=1, limit=10)

    assert result["meta"]["total"] >= 1
    assert result["data"][0]["id"] == "course-python-1"


@pytest.mark.asyncio
async def test_catalog_driven_typo_correction_matches_school_subjects(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(SCIENCE_COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-chem-1", "similarityScore": 0.91, "payload": {}},
                {"courseId": "course-physics-1", "similarityScore": 0.52, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "chemstry", {}, page=1, limit=10)

    assert result["meta"]["total"] >= 1
    assert result["data"][0]["id"] == "course-chem-1"
    assert result["data"][0]["subjectName"] == "Chemistry"


@pytest.mark.asyncio
async def test_partial_subject_typo_prefers_chemistry_instead_of_unrelated_catalog_terms(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(SCIENCE_COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-chem-1", "similarityScore": 0.89, "payload": {}},
                {"courseId": "course-physics-1", "similarityScore": 0.42, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "chestr", {}, page=1, limit=10)

    assert result["meta"]["total"] >= 1
    assert result["data"][0]["id"] == "course-chem-1"
    assert result["data"][0]["title"] == "Secondary Chemistry Exam Prep"


@pytest.mark.asyncio
async def test_jumbled_short_subject_typo_prefers_physics(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(SCIENCE_COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-physics-1", "similarityScore": 0.9, "payload": {}},
                {"courseId": "course-chem-1", "similarityScore": 0.35, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "phsyi", {}, page=1, limit=10)

    assert result["meta"]["total"] >= 1
    assert result["data"][0]["id"] == "course-physics-1"
    assert result["data"][0]["subjectName"] == "Physics"


@pytest.mark.asyncio
async def test_longer_physics_typo_prefers_physics(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(SCIENCE_COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-physics-1", "similarityScore": 0.91, "payload": {}},
                {"courseId": "course-chem-1", "similarityScore": 0.31, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "phyiscis", {}, page=1, limit=10)

    assert result["meta"]["total"] >= 1
    assert result["data"][0]["id"] == "course-physics-1"
    assert result["data"][0]["subjectName"] == "Physics"


@pytest.mark.asyncio
async def test_autocomplete_catalog_typo_correction_surfaces_subject_and_course(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(SCIENCE_COURSES))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-physics-1", "similarityScore": 0.88, "payload": {}},
            ]
        ),
    )

    result = await course_search_service.course_autocomplete("user-1", "phyics", limit=8)

    assert result[0]["type"] == "course"
    assert result[0]["courseId"] == "course-physics-1"
    assert any(item["type"] == "subject" and item["subjectName"] == "Physics" for item in result)


@pytest.mark.asyncio
async def test_mixed_query_prefers_results_covering_multiple_tokens(monkeypatch):
    mixed_courses = [
        {
            **COURSES[0],
            "id": "course-python-dev-1",
            "title": "Python Backend Development Bootcamp",
            "description": "Build backend development projects in Python APIs and services.",
            "subjectName": "Fullstack Development",
        },
        {
            **COURSES[1],
            "id": "course-de-1",
            "title": "Complete Spark Fundamentals Workshop",
            "description": "Learn spark fundamentals and streaming pipelines.",
            "subjectName": "Data Engineering",
        },
    ]
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(mixed_courses))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-de-1", "similarityScore": 0.66, "payload": {}},
                {"courseId": "course-python-dev-1", "similarityScore": 0.62, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "pyth dev", {}, page=1, limit=10)

    assert result["data"][0]["id"] == "course-python-dev-1"


@pytest.mark.asyncio
async def test_typo_python_query_normalizes_toward_python_courses(monkeypatch):
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(COURSES))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-web-1", "similarityScore": 0.72, "payload": {}},
                {"courseId": "course-python-1", "similarityScore": 0.69, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "pyhten", {}, page=1, limit=10)

    assert result["data"][0]["id"] == "course-python-1"


@pytest.mark.asyncio
async def test_shorthand_python_dev_query_normalizes_toward_python_development(monkeypatch):
    mixed_courses = [
        {
            **COURSES[0],
            "id": "course-python-dev-2",
            "title": "Python Development Workshop",
            "description": "Practical python development for backend apps.",
            "subjectName": "Fullstack Development",
        },
        {
            **COURSES[1],
            "id": "course-bi-1",
            "title": "Deep Dive Business Intelligence Workshop",
            "description": "Business intelligence analytics pipelines and reporting.",
            "subjectName": "Data Science",
        },
    ]
    monkeypatch.setattr(course_search_service.course_client, "get_all_courses", lambda: _async_value(mixed_courses))
    monkeypatch.setattr(course_search_service.course_client, "get_user_analytics_profile", lambda _user_id: _async_value({}))
    monkeypatch.setattr(course_search_service.embedding_service, "embed_text", lambda *_args, **_kwargs: _async_value([0.1, 0.2, 0.3]))
    monkeypatch.setattr(
        course_search_service,
        "vector_store",
        FakeVectorStore(
            [
                {"courseId": "course-bi-1", "similarityScore": 0.65, "payload": {}},
                {"courseId": "course-python-dev-2", "similarityScore": 0.61, "payload": {}},
            ]
        ),
    )
    monkeypatch.setattr(course_search_service, "_get_cluster_affinity", lambda _user_id: {})

    result = await course_search_service.semantic_course_search("user-1", "pyht dev", {}, page=1, limit=10)

    assert result["data"][0]["id"] == "course-python-dev-2"


@pytest.mark.asyncio
async def test_search_feedback_boosts_include_persistent_aggregate(monkeypatch):
    fake_cache = {}

    async def fake_get(key):
        return fake_cache.get(key)

    async def fake_setex(key, ttl, value):
        fake_cache[key] = value

    class FakeAggregateQuery:
        def __init__(self, rows):
            self.rows = rows

        def filter(self, *args, **kwargs):
            return self

        def all(self):
            return self.rows

    class FakeDb:
        def query(self, model):
            if model is course_search_service.SearchFeedbackAggregate:
                return FakeAggregateQuery(
                    [
                        SimpleNamespace(course_id="course-python-1", total_score=8.0),
                        SimpleNamespace(course_id="course-web-1", total_score=2.0),
                    ]
                )
            return FakeAggregateQuery([])

        def close(self):
            return None

    monkeypatch.setattr(course_search_service.redis_conn, "get", fake_get)
    monkeypatch.setattr(course_search_service.redis_conn, "setex", fake_setex)
    monkeypatch.setattr(course_search_service, "SessionLocal", lambda: FakeDb())

    boosts = await course_search_service.get_search_feedback_boosts("python")

    assert boosts["course-python-1"] == 1.0
    assert boosts["course-web-1"] == 0.25


def test_top_clicked_queries_analytics_projection(monkeypatch):
    row = SimpleNamespace(
        display_query="python",
        total_searches=10,
        zero_result_searches=2,
        total_clicks=6,
        total_previews=3,
        total_watches=2,
        total_enrolls=1,
        last_searched_at=None,
        last_feedback_at=None,
    )

    class FakeQuery:
        def order_by(self, *args, **kwargs):
            return self

        def limit(self, _limit):
            return self

        def all(self):
            return [row]

    class FakeDb:
        def query(self, _model):
            return FakeQuery()

        def close(self):
            return None

    monkeypatch.setattr(course_search_service, "SessionLocal", lambda: FakeDb())

    data = course_search_service.get_top_clicked_queries(limit=5)

    assert data[0]["query"] == "python"
    assert data[0]["zeroResultRate"] == 0.2


def test_zero_result_queries_analytics_projection(monkeypatch):
    row = SimpleNamespace(
        display_query="figm",
        total_searches=4,
        zero_result_searches=4,
        total_clicks=0,
        total_previews=0,
        total_watches=0,
        total_enrolls=0,
        last_searched_at=None,
        last_feedback_at=None,
    )

    class FakeQuery:
        def filter(self, *args, **kwargs):
            return self

        def order_by(self, *args, **kwargs):
            return self

        def limit(self, _limit):
            return self

        def all(self):
            return [row]

    class FakeDb:
        def query(self, _model):
            return FakeQuery()

        def close(self):
            return None

    monkeypatch.setattr(course_search_service, "SessionLocal", lambda: FakeDb())

    data = course_search_service.get_zero_result_queries(limit=5)

    assert data[0]["query"] == "figm"
    assert data[0]["zeroResultRate"] == 1.0


async def _async_value(value):
    return value
