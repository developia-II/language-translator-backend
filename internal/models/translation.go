package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Translation struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID         primitive.ObjectID `json:"userId" bson:"userId"`
	SourceText     string             `json:"sourceText" bson:"sourceText" validate:"required"`
	TranslatedText string             `json:"translatedText" bson:"translatedText"`
	SourceLang     string             `json:"sourceLang" bson:"sourceLang" validate:"required"`
	TargetLang     string             `json:"targetLang" bson:"targetLang" validate:"required"`
	CreatedAt      time.Time          `json:"createdAt" bson:"createdAt"`
}

type TranslateRequest struct {
	SourceText string `json:"sourceText" validate:"required"`
	SourceLang string `json:"sourceLang" validate:"required"`
	TargetLang string `json:"targetLang" validate:"required"`
}

type TranslateResponse struct {
	Translation Translation `json:"translation"`
}
