CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT UNIQUE,
  password BLOB,
  salt BLOB,
  github_id TEXT UNIQUE,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  verified INTEGER NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  userID INTEGER NOT NULL,
  expiry INTEGER NOT NULL,
  FOREIGN KEY (userID) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS verification_tokens (
  token TEXT PRIMARY KEY,
  userID INTEGER NOT NULL,
  expiry INTEGER NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (userID) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
  token TEXT PRIMARY KEY,
  userID INTEGER NOT NULL,
  expiry INTEGER NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (userID) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_name TEXT NOT NULL,
  userID INTEGER NOT NULL,
  text TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  reply_to INTEGER,
  original_reply_to INTEGER,
  FOREIGN KEY (userID) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (reply_to) REFERENCES comments(id) ON DELETE SET NULL,
  FOREIGN KEY (original_reply_to) REFERENCES comments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_comment_post_timestamp ON comments (post_name, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comment_replies ON comments (original_reply_to);

INSERT INTO users (id, username, name, email, verified) VALUES (1, "ashbythorpe", "Ashby Thorpe", "ashbythorpe@gmail.com", TRUE);
INSERT INTO comments (id, post_name, userID, text) VALUES (1, "example", 1, "A comment");
INSERT INTO comments (id, post_name, userID, text, reply_to, original_reply_to) VALUES (2, "example", 1, "A reply", 1, 1);
INSERT INTO comments (id, post_name, userID, text, reply_to, original_reply_to) VALUES (3, "example", 1, "A reply", 2, 1);
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
INSERT INTO comments (post_name, userID, text) VALUES ("example", 1, "A comment");
