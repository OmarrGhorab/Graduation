-- Migration: 000010_create_audit_logs.up.sql
-- Creates immutable audit_logs table for tracking all state changes

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- What changed
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    
    -- Who made the change
    actor_id UUID,
    actor_role VARCHAR(50),
    
    -- Change details
    old_value JSONB,
    new_value JSONB,
    
    -- Request context
    ip_address VARCHAR(45),
    user_agent TEXT,
    device_id VARCHAR(255),
    trace_id VARCHAR(64),
    
    -- Timestamp (immutable)
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for querying audit trail
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs(actor_id);
CREATE INDEX idx_audit_logs_occurred_at ON audit_logs(occurred_at);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);

-- Prevent updates and deletes on audit_logs (immutable)
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit logs are immutable and cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_audit_update
    BEFORE UPDATE ON audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER prevent_audit_delete
    BEFORE DELETE ON audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();
