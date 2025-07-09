CREATE TABLE IF NOT EXISTS oauth2_tokens (
                                             id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    access_token VARCHAR(512) UNIQUE NOT NULL,
    refresh_token VARCHAR(512),
    client_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255),
    scope TEXT,
    access_expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    refresh_expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_access_token ON oauth2_tokens(access_token);
CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_refresh_token ON oauth2_tokens(refresh_token);
CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_client_id ON oauth2_tokens(client_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_user_id ON oauth2_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_tokens_expires_at ON oauth2_tokens(access_expires_at);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_oauth2_tokens_updated_at
    BEFORE UPDATE ON oauth2_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
