package database

import "database/sql"

const ConversationsSchema = `
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    session_id TEXT,
    model TEXT NOT NULL,
    messages JSON NOT NULL,
    finish_reason TEXT,
    metadata JSON,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_conversations_session_id ON conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at);
`

const ServiceMetadataSchema = `
CREATE TABLE IF NOT EXISTS service_metadata (
    key TEXT PRIMARY KEY,
    value TEXT
);
`

func InitializeSchema(db *sql.DB) error {
	if _, err := db.Exec(ServiceMetadataSchema); err != nil {
		return err
	}
	if _, err := db.Exec(ConversationsSchema); err != nil {
		return err
	}
	// TODO: Add prompt_versions table for prompt A/B testing (see docs/EXTENDING.md)
	// TODO: Add fine_tune_jobs table for fine-tuning management
	// TODO: Add usage_logs table for cost tracking
	return nil
}
