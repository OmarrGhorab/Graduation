package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Enums
type ConversationType string

const (
	Direct ConversationType = "DIRECT"
	Group  ConversationType = "GROUP"
)

type MemberRole string

const (
	RoleOwner  MemberRole = "OWNER"
	RoleAdmin  MemberRole = "ADMIN"
	RoleMember MemberRole = "MEMBER"
)

type MessageType string

const (
	Text  MessageType = "text"
	Image MessageType = "image"
	Voice MessageType = "voice"
)

// Conversation
type Conversation struct {
	ID          string           `gorm:"type:uuid;primaryKey" json:"id"`
	Type        ConversationType `gorm:"type:varchar(20);not null" json:"type"`
	Name        string           `gorm:"type:varchar(255)" json:"name,omitempty"` // For groups
	Description string           `gorm:"type:text" json:"description,omitempty"`   // For groups
	ImageURL    string           `gorm:"type:varchar(255)" json:"image_url,omitempty"`
	CreatedBy   string           `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`

	// Relations
	Members  []ConversationMember `json:"members,omitempty"`
	Messages []Message            `json:"messages,omitempty"`
}

func (c *Conversation) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return
}

// ConversationMember
type ConversationMember struct {
	ConversationID string     `gorm:"type:uuid;primaryKey" json:"conversation_id"`
	UserID         string     `gorm:"type:uuid;primaryKey" json:"user_id"`
	Role           MemberRole `gorm:"type:varchar(20);default:'MEMBER'" json:"role"`
	JoinedAt       time.Time  `json:"joined_at"`
	LastReadAt     time.Time  `json:"last_read_at"`
}

// Message
type Message struct {
	ID             string         `gorm:"type:uuid;primaryKey" json:"id"`
	ConversationID string         `gorm:"type:uuid;not null;index" json:"conversation_id"`
	SenderID       string         `gorm:"type:uuid;not null" json:"sender_id"`
	Content        string         `gorm:"type:text" json:"content"`
	Type           MessageType    `gorm:"type:varchar(20);default:'text'" json:"type"`
	MediaURLs      pq.StringArray `gorm:"type:text[]" json:"media_urls,omitempty"` // Postgres Array
	ReplyToID      *string        `gorm:"type:uuid" json:"reply_to_id,omitempty"`
	IsDeleted      bool           `gorm:"default:false" json:"is_deleted"`
	CreatedAt      time.Time      `json:"created_at"`

	// Relations
	Sender *UserStub `gorm:"-" json:"sender,omitempty"` // Populated transiently
}

func (m *Message) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return
}

// Pinned Message
type PinnedMessage struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	MessageID      string    `gorm:"type:uuid;not null;uniqueIndex" json:"message_id"`
	ConversationID string    `gorm:"type:uuid;not null;index" json:"conversation_id"`
	PinnedBy       string    `gorm:"type:uuid;not null" json:"pinned_by"`
	PinnedAt       time.Time `gorm:"autoCreateTime" json:"pinned_at"`

	// Relation
	Message *Message `gorm:"foreignKey:MessageID" json:"message,omitempty"`
}

func (pm *PinnedMessage) BeforeCreate(tx *gorm.DB) (err error) {
	if pm.ID == "" {
		pm.ID = uuid.New().String()
	}
	return
}

// Device Token (For Push Notifications)
type DeviceToken struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"user_id"`
	Token     string    `gorm:"type:text;not null;uniqueIndex" json:"token"`
	Platform  string    `gorm:"type:varchar(20);default:'android'" json:"platform"` // android, ios, web
	UpdatedAt time.Time `json:"updated_at"`
}

func (dt *DeviceToken) BeforeCreate(tx *gorm.DB) (err error) {
	if dt.ID == "" {
		dt.ID = uuid.New().String()
	}
	return
}

// Read Receipt
type ReadReceipt struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	MessageID string    `gorm:"type:uuid;not null;index" json:"message_id"`
	UserID    string    `gorm:"type:uuid;not null" json:"user_id"`
	ReadAt    time.Time `gorm:"autoCreateTime" json:"read_at"`
}

func (rr *ReadReceipt) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.ID == "" {
		rr.ID = uuid.New().String()
	}
	return
}

// Basic User Stub
type UserStub struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
}

// Enriched Conversation Response
type ConversationResponse struct {
	ID          string           `json:"id"`
	Type        ConversationType `json:"type"`
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	ImageURL    string           `json:"image_url,omitempty"`
	CreatedBy   string           `json:"created_by"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`

	// Enriched fields
	Members      []ConversationMemberResponse `json:"members,omitempty"`
	LastMessage  *MessageResponse             `json:"last_message,omitempty"`
	PeerProfile  *UserStub                    `json:"peer_profile,omitempty"` // For direct chats
	PeerOnline   bool                         `json:"peer_online"`            // For direct chats (no omitempty so false shows)
	UnreadCount  int                          `json:"unread_count"`           // Number of unread messages
}

// Enriched Member Response
type ConversationMemberResponse struct {
	ConversationID string     `json:"conversation_id"`
	UserID         string     `json:"user_id"`
	Role           MemberRole `json:"role"`
	JoinedAt       time.Time  `json:"joined_at"`
	LastReadAt     time.Time  `json:"last_read_at"`
	
	// Enriched user profile
	Profile  *UserStub `json:"profile,omitempty"`
	IsOnline bool      `json:"is_online"`
}

// Enriched Message Response
type MessageResponse struct {
	ID             string         `json:"id"`
	ConversationID string         `json:"conversation_id"`
	SenderID       string         `json:"sender_id"`
	Content        string         `json:"content"`
	Type           MessageType    `json:"type"`
	MediaURLs      pq.StringArray `json:"media_urls,omitempty"`
	ReplyToID      *string        `json:"reply_to_id,omitempty"`
	IsDeleted      bool           `json:"is_deleted"`
	CreatedAt      time.Time      `json:"created_at"`
	
	// Enriched sender profile
	Sender  *UserStub              `json:"sender,omitempty"`
	ReplyTo *MessageReplyToStub    `json:"reply_to,omitempty"`
	
	// Read receipt info
	ReadBy  []ReadReceiptInfo      `json:"read_by,omitempty"`  // Who read this message (for groups)
	IsRead  bool                   `json:"is_read"`            // For direct chats - if peer read it
}

// Read Receipt Info
type ReadReceiptInfo struct {
	UserID string    `json:"user_id"`
	User   *UserStub `json:"user,omitempty"`
	ReadAt time.Time `json:"read_at"`
}

// Message Reply To Stub (for nested reply information)
type MessageReplyToStub struct {
	ID      string    `json:"id"`
	Content string    `json:"content"`
	Sender  *UserStub `json:"sender,omitempty"`
}

// Enriched Pinned Message Response
type PinnedMessageResponse struct {
	ID             string           `json:"id"`
	MessageID      string           `json:"message_id"`
	ConversationID string           `json:"conversation_id"`
	PinnedBy       string           `json:"pinned_by"`
	PinnedAt       time.Time        `json:"pinned_at"`
	
	// Enriched message with sender details
	Message *MessageResponse `json:"message,omitempty"`
	
	// Enriched pinner profile
	Pinner *UserStub `json:"pinner,omitempty"`
}

// Media Collection Response
type MediaCollectionResponse struct {
	Photos []MediaItem `json:"photos"`
	Voice  []MediaItem `json:"voice"`
	Links  []MediaItem `json:"links"`
}

// Media Item
type MediaItem struct {
	MessageID string         `json:"message_id"`
	URL       string         `json:"url"`
	Type      MessageType    `json:"type"`
	SenderID  string         `json:"sender_id"`
	Sender    *UserStub      `json:"sender,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}
