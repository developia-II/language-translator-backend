package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Conversation struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	Title     string             `json:"title" bson:"title"`
	Messages  []Message          `json:"messages" bson:"messages"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

type Message struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Role      string             `json:"role" bson:"role"` // "user" or "assistant"
	Content   string             `json:"content" bson:"content"`
	Language  string             `json:"language" bson:"language"` // Language code (en, yo, ig, ha)
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

type ChatRequest struct {
	ConversationID string `json:"conversationId,omitempty"` // Empty for new conversation
	Message        string `json:"message" validate:"required"`
	Language       string `json:"language" validate:"required"` // Target language for response
}

type ChatResponse struct {
	ConversationID string        `json:"conversationId"`
	Message        Message       `json:"message"`
	Conversation   *Conversation `json:"conversation,omitempty"`
}
