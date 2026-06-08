from app.clustering.cluster_service import ClusterService


class FakeQuery:
    def __init__(self, rows):
        self.rows = rows

    def filter(self, *args, **kwargs):
        return self

    def first(self):
        return None

    def all(self):
        return self.rows


class FakeDB:
    def __init__(self):
        self.added = []
        self.committed = False

    def query(self, model):
        return FakeQuery([])

    def add(self, row):
        self.added.append(row)

    def commit(self):
        self.committed = True


def test_cluster_service_reads_assignments():
    db = FakeDB()
    service = ClusterService(db)
    assert service.get_user_cluster("missing") is None
