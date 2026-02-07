-- Add read receipts and unread count support
-- Migration 002: Read Receipts and Notifications

-- Add last_read_at to conversation_members if not exists
ALTER TABLE conversation_members 
ADD COLUMN IF NOT EXISTS last_read_at TIMESTAMPTZ DEFAULT NOW();

-- Create index for efficient unread count queries
CREATE INDEX IF NOT EXISTS idx_members_last_read 
ON conversation_members(conversation_id, user_id, last_read_at);

-- Add image_url to conversations for group avatars
ALTER TABLE conversations
ADD COLUMN IF NOT EXISTS image_url TEXT;

-- Create read_receipts table for tracking who read which message
CREATE TABLE IF NOT EXISTS read_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    read_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(message_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_read_receipts_message ON read_receipts(message_id);
CREATE INDEX IF NOT EXISTS idx_read_receipts_user ON read_receipts(user_id);

-- Add delivered_at and read_at to messages for direct chat indicators
ALTER TABLE messages
ADD COLUMN IF NOT EXISTS delivered_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS read_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(conversation_id, delivered_at, read_at);
