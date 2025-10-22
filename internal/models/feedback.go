package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Feedback struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TranslationID primitive.ObjectID `json:"translationId" bson:"translationId" validate:"required"`
	UserID        primitive.ObjectID `json:"userId" bson:"userId"`
	Rating        int                `json:"rating" bson:"rating" validate:"required,min=1,max=5"`
	SuggestedText string             `json:"suggestedText,omitempty" bson:"suggestedText,omitempty"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
}

type FeedbackRequest struct {
	TranslationID string `json:"translationId" validate:"required"`
	Rating        int    `json:"rating" validate:"required,min=1,max=5"`
	SuggestedText string `json:"suggestedText,omitempty"`
}
