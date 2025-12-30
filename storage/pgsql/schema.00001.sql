CREATE INDEX idx_enrollment_queue_active_retrieval
ON enrollment_queue (id, priority DESC, created_at)
WHERE active = TRUE;

CREATE INDEX idx_command_results_lookup
ON command_results (id, command_uuid)
INCLUDE (status, updated_at);
