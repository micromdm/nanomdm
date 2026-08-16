ALTER TABLE enrollment_queue ADD INDEX idx_queue_lookup (id, active, priority DESC, created_at);
ALTER TABLE cert_auth_associations ADD INDEX idx_sha256 (sha256);
