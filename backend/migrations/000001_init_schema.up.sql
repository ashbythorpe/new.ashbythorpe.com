CREATE TABLE IF NOT EXISTS users (
    id BLOB PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    github_id TEXT UNIQUE,
    verified INTEGER NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS user_passwords (
    user_id BLOB PRIMARY KEY,
    password BLOB NOT NULL,
    salt BLOB NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user_oauth (
    provider TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    user_id BLOB NOT NULL,

    PRIMARY KEY (provider, provider_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sessions (
    id BLOB PRIMARY KEY,
    user_id BLOB NOT NULL,
    expiry INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS verification_tokens (
    token BLOB PRIMARY KEY,
    user_id BLOB NOT NULL,
    expiry INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token BLOB PRIMARY KEY,
    user_id BLOB NOT NULL,
    expiry INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_name TEXT NOT NULL,
    user_id BLOB NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    reply_to INTEGER,
    original_reply_to INTEGER,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (reply_to) REFERENCES comments (id) ON DELETE SET NULL,
    FOREIGN KEY (original_reply_to) REFERENCES comments (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_comment_post_timestamp ON comments (
    post_name, created_at DESC
);
CREATE INDEX IF NOT EXISTS idx_comment_replies ON comments (original_reply_to);

INSERT INTO users (id, name, email, verified) VALUES (
    1, 'Ashby Thorpe', 'ashbythorpe@gmail.com', TRUE
);
INSERT INTO comments (id, post_name, user_id, content) VALUES (
    1, 'example', 1, 'A comment'
);
