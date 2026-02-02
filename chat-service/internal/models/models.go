package models

import (
	"encoding/json"
	"time"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRoleStudent    UserRole = "STUDENT"
	UserRoleInstructor UserRole = "INSTRUCTOR"
	UserRoleTeacher    UserRole = "TEACHER"
	UserRoleParent     UserRole = "PARENT"
	UserRoleAssistant  UserRole = "ASSISTANT"
)

// MemberRole represents the role of a member within a conversation
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "OWNER"
	MemberRoleAdmin  MemberRole = "ADMIN"
	MemberRoleMember MemberRole = "MEMBER"
)

// ConversationType represents the type of conversation
type ConversationType string

const (
	ConversationTypeDirect ConversationType = "DIRECT"
	ConversationTypeGroup  ConversationType = "GROUP"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeImage  MessageType = "image"
	MessageTypeVoice  MessageType = "voice"
	MessageTypeSystem MessageType = "system"
)

// Conversation represents a chat conversation (direct or group)
type Conversation struct {
	ID          string               `gorm:"type:uuid;primaryKey" json:"id"`
	Type        ConversationType     `gorm:"type:varchar(20);not null" json:"type"`
	Name        string               `gorm:"type:varchar(255)" json:"name,omitempty"`
	Description string               `gorm:"type:text" json:"description,omitempty"`
	ImageURL    string               `gorm:"type:varchar(255)" json:"image_url,omitempty"`
	CreatedBy   string               `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt   time.Time            `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time            `gorm:"autoUpdateTime" json:"updated_at"`
	Members     []ConversationMember `gorm:"foreignKey:ConversationID" json:"members,omitempty"`

	// Transient fields
	UnreadCount         int    `gorm:"-" json:"unread_count"`
	LastMessageContent  string `gorm:"->" json:"-"`
	LastMessageSenderID string `gorm:"->" json:"-"`
	PreviewText         string `gorm:"-" json:"preview_text"`
}

// TableName specifies the table name for GORM
func (Conversation) TableName() string {
	return "conversations"
}

// ConversationMember represents a member of a conversation
type ConversationMember struct {
	ID                string     `gorm:"type:uuid;primaryKey" json:"id"`
	ConversationID    string     `gorm:"type:uuid;not null" json:"conversation_id"`
	UserID            string     `gorm:"type:uuid;not null" json:"user_id"`
	UserRole          UserRole   `gorm:"type:varchar(20);not null" json:"user_role"`
	MemberRole        MemberRole `gorm:"type:varchar(20);not null;default:'MEMBER'" json:"member_role"`
	JoinedAt          time.Time  `gorm:"autoCreateTime" json:"joined_at"`
	LeftAt            *time.Time `json:"left_at,omitempty"`
	LastReadAt        *time.Time `json:"last_read_at,omitempty"`
	LastReadMessageID *string    `gorm:"type:uuid" json:"last_read_message_id,omitempty"`
	UnreadCount       int        `gorm:"default:0" json:"unread_count"`

	// Transient fields
	UserName  string `gorm:"-" json:"user_name,omitempty"`
	UserImage string `gorm:"-" json:"user_image,omitempty"`
}

// TableName specifies the table name for GORM
func (ConversationMember) TableName() string {
	return "conversation_members"
}

// Message represents a chat message
type Message struct {
	ID             string          `gorm:"type:uuid;primaryKey" json:"id"`
	LocalID        string          `gorm:"type:uuid;index" json:"local_id"` // Client-side ID for idempotency
	ConversationID string          `gorm:"type:uuid;not null" json:"conversation_id"`
	SenderID       string          `gorm:"type:uuid;not null" json:"sender_id"`
	SenderRole     UserRole        `gorm:"type:varchar(20);not null" json:"sender_role"`
	Type           MessageType     `gorm:"type:varchar(20);not null;default:'text'" json:"type"`
	Content        string          `gorm:"type:text" json:"content,omitempty"`
	MediaURLs      []string        `gorm:"type:text[]" json:"media_urls,omitempty"`
	MediaMetadata  json.RawMessage `gorm:"type:jsonb" json:"media_metadata,omitempty"`
	ReplyToID      *string         `gorm:"type:uuid" json:"reply_to_id,omitempty"`
	ReplyTo        *Message        `gorm:"foreignKey:ReplyToID" json:"reply_to,omitempty"`
	IsDeleted      bool            `gorm:"default:false" json:"is_deleted"`
	CreatedAt      time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	// Transient fields
	SenderName  string `gorm:"-" json:"sender_name,omitempty"`
	SenderImage string `gorm:"-" json:"sender_image,omitempty"`
}

// TableName specifies the table name for GORM
func (Message) TableName() string {
	return "messages"
}

// PinnedMessage represents a pinned message in a conversation
type PinnedMessage struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	MessageID      string    `gorm:"type:uuid;not null;uniqueIndex" json:"message_id"`
	ConversationID string    `gorm:"type:uuid;not null" json:"conversation_id"`
	PinnedBy       string    `gorm:"type:uuid;not null" json:"pinned_by"`
	PinnedAt       time.Time `gorm:"autoCreateTime" json:"pinned_at"`
	Message        *Message  `gorm:"foreignKey:MessageID" json:"message,omitempty"`
}

// TableName specifies the table name for GORM
func (PinnedMessage) TableName() string {
	return "pinned_messages"
}

// DeviceToken represents a push notification token for a device
type DeviceToken struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    string    `gorm:"type:uuid;not null" json:"user_id"`
	Token     string    `gorm:"type:text;not null;uniqueIndex" json:"token"`
	Platform  string    `gorm:"type:varchar(20);not null" json:"platform"` // ios, android
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (DeviceToken) TableName() string {
	return "device_tokens"
}
