CREATE EXTENSION IF NOT EXISTS pgcrypto;

UPDATE sessions SET id = encode(digest(id, 'sha256'), 'hex');

UPDATE machines SET token = encode(digest(token, 'sha256'), 'hex');

UPDATE accounts
SET confirm_token = encode(digest(confirm_token, 'sha256'), 'hex')
WHERE confirm_token IS NOT NULL AND confirm_token <> '';

UPDATE accounts
SET reset_token = encode(digest(reset_token, 'sha256'), 'hex')
WHERE reset_token IS NOT NULL AND reset_token <> '';

UPDATE spent_confirms SET token = encode(digest(token, 'sha256'), 'hex');
