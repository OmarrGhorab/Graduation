-- Chat Service Database Schema
-- PostgreSQL 14+

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enum Types
CREATE TYPE user_role AS ENUM ('STUDENT', 'INSTRUCTOR', 'TEACHER', 'PARENT', 'ASSISTANT');
CREATE TYPE conversation_type AS ENUM ('DIRECT', 'GROUP');
CREATE TYPE message_type AS ENUM ('text', 'image', 'voice');
CREATE TYPE member_role AS ENUM ('OWNER', 'ADMIN', 'MEMBER');

-- Conversations Table
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type conversation_type NOT NULL,
    name VARCHAR(255),           -- NULL for direct chats
    description TEXT,
    created_by UUID NOT NULL,    -- User ID from auth service
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_conversations_created_by ON conversations(created_by);
CREATE INDEX idx_conversations_type ON conversations(type);

-- Conversation Members Table
CREATE TABLE conversation_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    user_role user_role NOT NULL,        -- Role from auth service
    member_role member_role NOT NULL DEFAULT 'MEMBER',
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    left_at TIMESTAMPTZ,                 -- NULL if still active
    
    UNIQUE(conversation_id, user_id)
);

CREATE INDEX idx_members_conversation ON conversation_members(conversation_id);
CREATE INDEX idx_members_user ON conversation_members(user_id);
CREATE INDEX idx_members_active ON conversation_members(conversation_id) WHERE left_at IS NULL;

-- Messages Table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL,
    sender_role user_role NOT NULL,
    message_type message_type NOT NULL DEFAULT 'text',
    content TEXT,                        -- Text content
    media_url TEXT,                      -- For image/voice messages
    media_metadata JSONB,                -- {size, duration, mime_type}
    reply_to_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at DESC);
CREATE INDEX idx_messages_sender ON messages(sender_id);
CREATE INDEX idx_messages_reply_to ON messages(reply_to_id) WHERE reply_to_id IS NOT NULL;

-- Pinned Messages Table
CREATE TABLE pinned_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    pinned_by UUID NOT NULL,
    pinned_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(message_id)
);

CREATE INDEX idx_pinned_conversation ON pinned_messages(conversation_id);

-- Device Tokens for Push Notifications
CREATE TABLE device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    token TEXT NOT NULL,
    platform VARCHAR(20) NOT NULL,       -- 'ios' or 'android'
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(token)
);

CREATE INDEX idx_device_tokens_user ON device_tokens(user_id) WHERE is_active = TRUE;

-- Update timestamp trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update trigger to tables
CREATE TRIGGER update_conversations_updated_at
    BEFORE UPDATE ON conversations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_device_tokens_updated_at
    BEFORE UPDATE ON device_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
