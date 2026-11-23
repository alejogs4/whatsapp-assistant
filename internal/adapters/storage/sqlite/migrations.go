package sqlite

var schemaMigrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		phone_number TEXT NOT NULL UNIQUE,
		timezone TEXT NOT NULL,
		morning_catalyst_hour INTEGER NOT NULL,
		evening_reflection_hour INTEGER NOT NULL,
		daily_touches_used INTEGER NOT NULL DEFAULT 0,
		last_touch_sent_at TEXT
	);`,

	`CREATE TABLE IF NOT EXISTS learning_paths (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		title TEXT NOT NULL,
		source_folder_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`,
	`CREATE INDEX IF NOT EXISTS idx_learning_paths_user_id ON learning_paths(user_id);`,

	`CREATE TABLE IF NOT EXISTS mastery_ledger (
		user_id TEXT NOT NULL,
		path_id TEXT NOT NULL,
		concept_key TEXT NOT NULL,
		strength_score REAL NOT NULL DEFAULT 0.0,
		interval_days INTEGER NOT NULL DEFAULT 0,
		ease_factor REAL NOT NULL DEFAULT 2.5,
		next_review_at TEXT NOT NULL,
		misconceptions_json TEXT NOT NULL DEFAULT '[]',
		PRIMARY KEY (user_id, path_id, concept_key),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (path_id) REFERENCES learning_paths(id) ON DELETE CASCADE
	);`,
	`CREATE INDEX IF NOT EXISTS idx_mastery_ledger_due ON mastery_ledger(user_id, path_id, next_review_at);`,

	`CREATE TABLE IF NOT EXISTS touchpoint_sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		path_id TEXT NOT NULL,
		concept_key TEXT NOT NULL,
		state TEXT NOT NULL,
		touchpoint_type TEXT NOT NULL,
		prompt_text TEXT NOT NULL DEFAULT '',
		user_answer TEXT NOT NULL DEFAULT '',
		evaluation_feedback TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		completed_at TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (path_id) REFERENCES learning_paths(id) ON DELETE CASCADE
	);`,
	`CREATE INDEX IF NOT EXISTS idx_touchpoint_sessions_user_state ON touchpoint_sessions(user_id, state);`,
	`CREATE INDEX IF NOT EXISTS idx_touchpoint_sessions_user_created ON touchpoint_sessions(user_id, created_at);`,

	`CREATE TABLE IF NOT EXISTS learning_path_chunks (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		path_id TEXT NOT NULL,
		content TEXT NOT NULL,
		embedding_blob BLOB NOT NULL,
		created_at TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (path_id) REFERENCES learning_paths(id) ON DELETE CASCADE
	);`,
	`CREATE INDEX IF NOT EXISTS idx_chunks_user_path ON learning_path_chunks(user_id, path_id);`,
}
