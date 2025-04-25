CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(30) UNIQUE NOT NULL,
    email VARCHAR(255) NOT NULL
);

CREATE TABLE refresh_tokens (
    token_id VARCHAR(36) PRIMARY KEY,
    token_hash VARCHAR NOT NULL,
    expires_at TIMESTAMP NOT NULL
);
