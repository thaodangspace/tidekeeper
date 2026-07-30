DROP INDEX IF EXISTS accounts_active_email_idx;

ALTER TABLE accounts
    DROP CONSTRAINT IF EXISTS accounts_password_hash_check,
    DROP CONSTRAINT IF EXISTS accounts_email_normalized_check,
    DROP CONSTRAINT IF EXISTS accounts_email_key,
    DROP COLUMN IF EXISTS password_hash,
    DROP COLUMN IF EXISTS email;
