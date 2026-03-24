ALTER TABLE audit_events ADD COLUMN archived BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_audit_events_archived ON audit_events(archived, created_at);
