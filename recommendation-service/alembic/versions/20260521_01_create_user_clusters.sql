-- Recommendation Service Migration: clustering tables
-- Date: 2026-05-21

CREATE TABLE IF NOT EXISTS user_clusters (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL UNIQUE,
    cluster_id INTEGER NOT NULL,
    embedding_version VARCHAR(64) NOT NULL DEFAULT 'bge-small-en-v1.5',
    feature_version VARCHAR(32) NOT NULL DEFAULT 'v1',
    distance_to_centroid DOUBLE PRECISION,
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata_json JSONB
);

CREATE INDEX IF NOT EXISTS idx_user_clusters_user_id ON user_clusters(user_id);
CREATE INDEX IF NOT EXISTS idx_user_clusters_cluster_id ON user_clusters(cluster_id);

CREATE TABLE IF NOT EXISTS cluster_metadata (
    id VARCHAR(64) PRIMARY KEY,
    cluster_id INTEGER NOT NULL UNIQUE,
    label VARCHAR(128) NOT NULL DEFAULT 'general',
    centroid_vector JSONB,
    top_subjects JSONB,
    top_courses JSONB,
    user_count INTEGER NOT NULL DEFAULT 0,
    model_version VARCHAR(64) NOT NULL DEFAULT 'kmeans-v1',
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cluster_metadata_cluster_id ON cluster_metadata(cluster_id);
