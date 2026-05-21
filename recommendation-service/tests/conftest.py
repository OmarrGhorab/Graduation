import os
import sys
import types
import math


ROOT = os.path.dirname(os.path.dirname(__file__))
if ROOT not in sys.path:
    sys.path.insert(0, ROOT)

os.environ.setdefault("AI_API_KEY", "test-key")
os.environ.setdefault("DATABASE_URL", "sqlite:///./test.db")
os.environ.setdefault("INTERNAL_SERVICE_SECRET", "test-secret")
os.environ.setdefault("CLOUDINARY_CLOUD_NAME", "test-cloud")
os.environ.setdefault("CLOUDINARY_API_KEY", "test-cloud-key")
os.environ.setdefault("CLOUDINARY_API_SECRET", "test-cloud-secret")

if "redis" not in sys.modules:
    redis_stub = types.ModuleType("redis")
    redis_asyncio_stub = types.ModuleType("redis.asyncio")

    class _FakeRedis:
        def __init__(self):
            self.store = {}

        async def get(self, key):
            return self.store.get(key)

        async def setex(self, key, ttl, value):
            self.store[key] = value

        async def delete(self, key):
            self.store.pop(key, None)

        async def ping(self):
            return True

    def _from_url(*args, **kwargs):
        return _FakeRedis()

    redis_asyncio_stub.from_url = _from_url
    redis_stub.asyncio = redis_asyncio_stub
    sys.modules["redis"] = redis_stub
    sys.modules["redis.asyncio"] = redis_asyncio_stub

if "opentelemetry" not in sys.modules:
    opentelemetry_stub = types.ModuleType("opentelemetry")
    trace_stub = types.ModuleType("opentelemetry.trace")

    class _FakeSpan:
        def __enter__(self):
            return self

        def __exit__(self, exc_type, exc, tb):
            return False

        def set_attribute(self, *args, **kwargs):
            return None

    class _FakeTracer:
        def start_as_current_span(self, *args, **kwargs):
            return _FakeSpan()

    def _get_tracer(*args, **kwargs):
        return _FakeTracer()

    trace_stub.get_tracer = _get_tracer
    opentelemetry_stub.trace = trace_stub
    sys.modules["opentelemetry"] = opentelemetry_stub
    sys.modules["opentelemetry.trace"] = trace_stub

if "qdrant_client" not in sys.modules:
    qdrant_stub = types.ModuleType("qdrant_client")
    qdrant_http_stub = types.ModuleType("qdrant_client.http")
    qdrant_models_stub = types.ModuleType("qdrant_client.http.models")

    class _AsyncQdrantClient:
        def __init__(self, *args, **kwargs):
            pass

        async def collection_exists(self, *args, **kwargs):
            return True

        async def create_collection(self, *args, **kwargs):
            return None

        async def upsert(self, *args, **kwargs):
            return None

        async def search(self, *args, **kwargs):
            return []

    class _Distance:
        COSINE = "Cosine"

    class _VectorParams:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

    class _PointStruct:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

    class _MatchValue:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

    class _FieldCondition:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

    class _Filter:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

    qdrant_models_stub.Distance = _Distance
    qdrant_models_stub.VectorParams = _VectorParams
    qdrant_models_stub.PointStruct = _PointStruct
    qdrant_models_stub.MatchValue = _MatchValue
    qdrant_models_stub.FieldCondition = _FieldCondition
    qdrant_models_stub.Filter = _Filter
    qdrant_http_stub.models = qdrant_models_stub
    qdrant_stub.AsyncQdrantClient = _AsyncQdrantClient
    qdrant_stub.http = qdrant_http_stub
    sys.modules["qdrant_client"] = qdrant_stub
    sys.modules["qdrant_client.http"] = qdrant_http_stub
    sys.modules["qdrant_client.http.models"] = qdrant_models_stub

if "sentence_transformers" not in sys.modules:
    sentence_transformers_stub = types.ModuleType("sentence_transformers")

    class _FakeEncoded(list):
        def tolist(self):
            return list(self)

    class _FakeSentenceTransformer:
        def __init__(self, *args, **kwargs):
            pass

        def encode(self, texts, normalize_embeddings=True):
            return _FakeEncoded([[0.1, 0.2, 0.3] for _ in texts])

    sentence_transformers_stub.SentenceTransformer = _FakeSentenceTransformer
    sys.modules["sentence_transformers"] = sentence_transformers_stub

if "sqlalchemy" not in sys.modules:
    sqlalchemy_stub = types.ModuleType("sqlalchemy")
    sqlalchemy_orm_stub = types.ModuleType("sqlalchemy.orm")
    sqlalchemy_ext_stub = types.ModuleType("sqlalchemy.ext")
    sqlalchemy_declarative_stub = types.ModuleType("sqlalchemy.ext.declarative")
    sqlalchemy_dialects_stub = types.ModuleType("sqlalchemy.dialects")
    sqlalchemy_postgresql_stub = types.ModuleType("sqlalchemy.dialects.postgresql")

    class _FakeColumn:
        def __init__(self, *args, **kwargs):
            self.args = args
            self.kwargs = kwargs

        def __eq__(self, other):
            return ("eq", self, other)

    class _FakeType:
        def __init__(self, *args, **kwargs):
            pass

    class _FakeConnection:
        def __enter__(self):
            return self

        def __exit__(self, exc_type, exc, tb):
            return False

        def execute(self, *args, **kwargs):
            return None

    class _FakeEngine:
        def connect(self):
            return _FakeConnection()

    class _FakeBase:
        metadata = types.SimpleNamespace(create_all=lambda *args, **kwargs: None)

        def __init__(self, **kwargs):
            for key, value in kwargs.items():
                setattr(self, key, value)

    def _create_engine(*args, **kwargs):
        return _FakeEngine()

    def _sessionmaker(*args, **kwargs):
        class _SessionLocal:
            def __call__(self):
                return None

        return _SessionLocal()

    def _declarative_base():
        return _FakeBase

    sqlalchemy_stub.Column = _FakeColumn
    sqlalchemy_stub.Boolean = _FakeType
    sqlalchemy_stub.DateTime = _FakeType
    sqlalchemy_stub.Float = _FakeType
    sqlalchemy_stub.Integer = _FakeType
    sqlalchemy_stub.JSON = _FakeType
    sqlalchemy_stub.String = _FakeType
    sqlalchemy_stub.create_engine = _create_engine
    sqlalchemy_stub.text = lambda value: value
    sqlalchemy_orm_stub.Session = object
    sqlalchemy_orm_stub.sessionmaker = _sessionmaker
    sqlalchemy_declarative_stub.declarative_base = _declarative_base
    sqlalchemy_ext_stub.declarative = sqlalchemy_declarative_stub
    sqlalchemy_postgresql_stub.UUID = _FakeType
    sqlalchemy_dialects_stub.postgresql = sqlalchemy_postgresql_stub
    sys.modules["sqlalchemy"] = sqlalchemy_stub
    sys.modules["sqlalchemy.orm"] = sqlalchemy_orm_stub
    sys.modules["sqlalchemy.ext"] = sqlalchemy_ext_stub
    sys.modules["sqlalchemy.ext.declarative"] = sqlalchemy_declarative_stub
    sys.modules["sqlalchemy.dialects"] = sqlalchemy_dialects_stub
    sys.modules["sqlalchemy.dialects.postgresql"] = sqlalchemy_postgresql_stub

if "google.genai" not in sys.modules:
    google_stub = sys.modules.get("google") or types.ModuleType("google")
    genai_stub = types.ModuleType("google.genai")
    genai_types_stub = types.ModuleType("google.genai.types")

    class _FakeResponse:
        text = "[]"

    class _FakeModels:
        def generate_content(self, *args, **kwargs):
            return _FakeResponse()

        async def generate_content_stream(self, *args, **kwargs):
            if False:
                yield _FakeResponse()

    class _FakeClient:
        def __init__(self, *args, **kwargs):
            self.models = _FakeModels()
            self.aio = types.SimpleNamespace(models=_FakeModels())

    class _GenerateContentConfig:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

    class _Part:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

    class _Blob:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

    class _Content:
        def __init__(self, **kwargs):
            self.kwargs = kwargs

    genai_types_stub.GenerateContentConfig = _GenerateContentConfig
    genai_types_stub.Part = _Part
    genai_types_stub.Blob = _Blob
    genai_types_stub.Content = _Content
    genai_stub.Client = _FakeClient
    genai_stub.types = genai_types_stub
    google_stub.genai = genai_stub
    sys.modules["google"] = google_stub
    sys.modules["google.genai"] = genai_stub
    sys.modules["google.genai.types"] = genai_types_stub

if "numpy" not in sys.modules:
    numpy_stub = types.ModuleType("numpy")

    class _FakeArray(list):
        pass

    class _FakeLinalg:
        @staticmethod
        def norm(value):
            return 0.0

    def _array(value, dtype=None):
        if isinstance(value, list):
            return _FakeArray(value)
        return _FakeArray([value])

    numpy_stub.array = _array
    numpy_stub.linalg = _FakeLinalg()
    numpy_stub.ndarray = _FakeArray
    sys.modules["numpy"] = numpy_stub

if "sklearn" not in sys.modules:
    sklearn_stub = types.ModuleType("sklearn")
    cluster_stub = types.ModuleType("sklearn.cluster")

    class _FakeKMeans:
        def __init__(self, n_clusters=1, random_state=None, n_init=10):
            self.n_clusters = n_clusters
            self.cluster_centers_ = []

        def fit_predict(self, matrix):
            size = len(matrix) if hasattr(matrix, "__len__") else 1
            self.cluster_centers_ = [[0.0] * (len(matrix[0]) if size and hasattr(matrix[0], "__len__") else 1) for _ in range(self.n_clusters)]
            return [0 for _ in range(size)]

    cluster_stub.KMeans = _FakeKMeans
    sklearn_stub.cluster = cluster_stub
    sys.modules["sklearn"] = sklearn_stub
    sys.modules["sklearn.cluster"] = cluster_stub

if "langgraph" not in sys.modules:
    langgraph_stub = types.ModuleType("langgraph")
    graph_stub = types.ModuleType("langgraph.graph")

    END = "__end__"

    class _FakeStateGraph:
        def __init__(self, state_type):
            self.state_type = state_type

        def add_node(self, *args, **kwargs):
            return None

        def set_entry_point(self, *args, **kwargs):
            return None

        def add_conditional_edges(self, *args, **kwargs):
            return None

        def add_edge(self, *args, **kwargs):
            return None

        def compile(self):
            class _Compiled:
                async def ainvoke(self, state):
                    return state

            return _Compiled()

    graph_stub.END = END
    graph_stub.StateGraph = _FakeStateGraph
    langgraph_stub.graph = graph_stub
    sys.modules["langgraph"] = langgraph_stub
    sys.modules["langgraph.graph"] = graph_stub
