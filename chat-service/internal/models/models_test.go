package models

import (
	"testing"
)

func TestUserRole_Values(t *testing.T) {
	roles := []UserRole{
		UserRoleStudent,
		UserRoleInstructor,
		UserRoleTeacher,
		UserRoleParent,
		UserRoleAssistant,
	}

	expected := []string{"STUDENT", "INSTRUCTOR", "TEACHER", "PARENT", "ASSISTANT"}

	for i, role := range roles {
		if string(role) != expected[i] {
			t.Errorf("UserRole = %v, want %v", role, expected[i])
		}
	}
}

func TestMemberRole_Values(t *testing.T) {
	roles := []MemberRole{
		MemberRoleOwner,
		MemberRoleAdmin,
		MemberRoleMember,
	}

	expected := []string{"OWNER", "ADMIN", "MEMBER"}

	for i, role := range roles {
		if string(role) != expected[i] {
			t.Errorf("MemberRole = %v, want %v", role, expected[i])
		}
	}
}

func TestConversationType_Values(t *testing.T) {
	types := []ConversationType{
		ConversationTypeDirect,
		ConversationTypeGroup,
	}

	expected := []string{"DIRECT", "GROUP"}

	for i, ct := range types {
		if string(ct) != expected[i] {
			t.Errorf("ConversationType = %v, want %v", ct, expected[i])
		}
	}
}

func TestMessageType_Values(t *testing.T) {
	types := []MessageType{
		MessageTypeText,
		MessageTypeImage,
		MessageTypeVoice,
	}

	expected := []string{"text", "image", "voice"}

	for i, mt := range types {
		if string(mt) != expected[i] {
			t.Errorf("MessageType = %v, want %v", mt, expected[i])
		}
	}
}

func TestConversation_TableName(t *testing.T) {
	c := Conversation{}
	if c.TableName() != "conversations" {
		t.Errorf("TableName() = %v, want conversations", c.TableName())
	}
}

func TestConversationMember_TableName(t *testing.T) {
	cm := ConversationMember{}
	if cm.TableName() != "conversation_members" {
		t.Errorf("TableName() = %v, want conversation_members", cm.TableName())
	}
}

func TestMessage_TableName(t *testing.T) {
	m := Message{}
	if m.TableName() != "messages" {
		t.Errorf("TableName() = %v, want messages", m.TableName())
	}
}

func TestPinnedMessage_TableName(t *testing.T) {
	pm := PinnedMessage{}
	if pm.TableName() != "pinned_messages" {
		t.Errorf("TableName() = %v, want pinned_messages", pm.TableName())
	}
}

func TestDeviceToken_TableName(t *testing.T) {
	dt := DeviceToken{}
	if dt.TableName() != "device_tokens" {
		t.Errorf("TableName() = %v, want device_tokens", dt.TableName())
	}
}

func TestConversation_Struct(t *testing.T) {
	conv := Conversation{
		ID:        "test-id",
		Type:      ConversationTypeGroup,
		Name:      "Test Group",
		CreatedBy: "creator-id",
	}

	if conv.ID != "test-id" {
		t.Errorf("ID = %v, want test-id", conv.ID)
	}
	if conv.Type != ConversationTypeGroup {
		t.Errorf("Type = %v, want GROUP", conv.Type)
	}
	if conv.Name != "Test Group" {
		t.Errorf("Name = %v, want Test Group", conv.Name)
	}
	if conv.CreatedBy != "creator-id" {
		t.Errorf("CreatedBy = %v, want creator-id", conv.CreatedBy)
	}
}

func TestMessage_Struct(t *testing.T) {
	replyID := "reply-id"
	msg := Message{
		ID:             "msg-id",
		ConversationID: "conv-id",
		SenderID:       "sender-id",
		SenderRole:     UserRoleTeacher,
		Type:           MessageTypeText,
		Content:        "Hello world",
		ReplyToID:      &replyID,
		IsDeleted:      false,
	}

	if msg.Type != MessageTypeText {
		t.Errorf("Type = %v, want text", msg.Type)
	}
	if msg.SenderRole != UserRoleTeacher {
		t.Errorf("SenderRole = %v, want TEACHER", msg.SenderRole)
	}
	if *msg.ReplyToID != replyID {
		t.Errorf("ReplyToID = %v, want %v", *msg.ReplyToID, replyID)
	}
	if msg.ID != "msg-id" {
		t.Errorf("ID = %v, want msg-id", msg.ID)
	}
	if msg.ConversationID != "conv-id" {
		t.Errorf("ConversationID = %v, want conv-id", msg.ConversationID)
	}
	if msg.SenderID != "sender-id" {
		t.Errorf("SenderID = %v, want sender-id", msg.SenderID)
	}
	if msg.Content != "Hello world" {
		t.Errorf("Content = %v, want Hello world", msg.Content)
	}
	if msg.IsDeleted {
		t.Error("IsDeleted should be false")
	}
}

func TestConversationMember_Struct(t *testing.T) {
	member := ConversationMember{
		ID:             "member-id",
		ConversationID: "conv-id",
		UserID:         "user-id",
		UserRole:       UserRoleStudent,
		MemberRole:     MemberRoleMember,
	}

	if member.UserRole != UserRoleStudent {
		t.Errorf("UserRole = %v, want STUDENT", member.UserRole)
	}
	if member.MemberRole != MemberRoleMember {
		t.Errorf("MemberRole = %v, want MEMBER", member.MemberRole)
	}
	if member.ID != "member-id" {
		t.Errorf("ID = %v, want member-id", member.ID)
	}
	if member.ConversationID != "conv-id" {
		t.Errorf("ConversationID = %v, want conv-id", member.ConversationID)
	}
	if member.UserID != "user-id" {
		t.Errorf("UserID = %v, want user-id", member.UserID)
	}
}
