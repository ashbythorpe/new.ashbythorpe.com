CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    password BLOB,
    salt BLOB,
    github_id TEXT UNIQUE,
    verified INTEGER NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS sessions (
    id BLOB PRIMARY KEY,
    userid INTEGER NOT NULL,
    expiry INTEGER NOT NULL,
    FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS verification_tokens (
    token BLOB PRIMARY KEY,
    userid INTEGER NOT NULL,
    expiry INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token BLOB PRIMARY KEY,
    userid INTEGER NOT NULL,
    expiry INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_name TEXT NOT NULL,
    userid INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    reply_to INTEGER,
    original_reply_to INTEGER,
    FOREIGN KEY (userid) REFERENCES users (id) ON DELETE CASCADE,
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
INSERT INTO comments (id, post_name, userid, content) VALUES (
    1, 'example', 1, 'A comment'
);
