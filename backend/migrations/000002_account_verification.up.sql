-- Existing codes use a global hash and cannot be safely reused.
DELETE FROM verification_tokens;
ALTER TABLE verification_tokens ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0;
CREATE UNIQUE INDEX idx_verification_user ON verification_tokens (user_id);
