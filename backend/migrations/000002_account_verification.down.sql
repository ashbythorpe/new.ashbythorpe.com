DELETE FROM verification_tokens;
DROP INDEX idx_verification_user;
ALTER TABLE verification_tokens DROP COLUMN attempts;
