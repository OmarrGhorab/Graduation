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
