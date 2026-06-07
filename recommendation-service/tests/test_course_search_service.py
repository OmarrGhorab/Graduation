import pytest

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
    assert result["data"][0]["matchSource"] == "keyword_fallback"


async def _async_value(value):
    return value
