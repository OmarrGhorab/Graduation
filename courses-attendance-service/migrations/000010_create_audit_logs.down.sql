-- Migration: 000010_create_audit_logs.down.sql
DROP TRIGGER IF EXISTS prevent_audit_delete ON audit_logs;
DROP TRIGGER IF EXISTS prevent_audit_update ON audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_modification();
DROP TABLE IF EXISTS audit_logs;
