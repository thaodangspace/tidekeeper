-- First-party email/password credentials. Password hashes use the encoded Argon2id
-- format produced by the application; neither plaintext passwords nor session tokens
-- are ever persisted.
ALTER TABLE accounts
    ADD COLUMN email text NOT NULL,
    ADD COLUMN password_hash text NOT NULL,
    ADD CONSTRAINT accounts_email_key UNIQUE (email),
    ADD CONSTRAINT accounts_email_normalized_check
        CHECK (email = lower(email)
            AND char_length(email) BETWEEN 3 AND 254
            AND email ~ '^[^[:space:]@]+@[^[:space:]@]+\.[^[:space:]@]+$'),
    ADD CONSTRAINT accounts_password_hash_check
        CHECK (char_length(password_hash) BETWEEN 1 AND 512);

CREATE INDEX accounts_active_email_idx
    ON accounts (email)
    WHERE status = 'ACTIVE';
