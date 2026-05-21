from pathlib import Path


def test_clustering_migration_has_apply_and_rollback_contract():
    migration = Path("alembic/versions/20260521_01_create_user_clusters.sql")
    sql = migration.read_text(encoding="utf-8").lower()

    assert "create table if not exists user_clusters" in sql
    assert "create table if not exists cluster_metadata" in sql
    assert "-- rollback:" in sql
    assert "drop table if exists cluster_metadata" in sql
    assert "drop table if exists user_clusters" in sql
