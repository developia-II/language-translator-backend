package handlers

import (
	"context"
	"time"

	"github.com/developia-II/language-translator-backend/internal/database"
	"github.com/developia-II/language-translator-backend/internal/models"
	"github.com/developia-II/language-translator-backend/internal/services"
	"github.com/developia-II/language-translator-backend/utils"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Translate(c *fiber.Ctx) error {
	var req models.TranslateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := utils.Validate.Struct(req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	// Get user ID from context
	userID := c.Locals("userId").(string)
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	// Call translation service
	translatedText, err := services.TranslateText(req.SourceText, req.SourceLang, req.TargetLang)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Translation failed: "+err.Error())
	}

	// Save translation to database
	translation := models.Translation{
		ID:             primitive.NewObjectID(),
		UserID:         userObjID,
		SourceText:     req.SourceText,
		TranslatedText: translatedText,
		SourceLang:     req.SourceLang,
		TargetLang:     req.TargetLang,
		CreatedAt:      time.Now(),
	}

	collection := database.GetCollection("translations")
	_, err = collection.InsertOne(context.Background(), translation)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to save translation")
	}

	return c.JSON(models.TranslateResponse{
		Translation: translation,
	})
}

func GetTranslations(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	collection := database.GetCollection("translations")

	// Find all translations for user, sorted by most recent
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(50)
	cursor, err := collection.Find(context.Background(), bson.M{"userId": userObjID}, opts)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch translations")
	}
	defer cursor.Close(context.Background())

	var translations []models.Translation
	if err := cursor.All(context.Background(), &translations); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to decode translations")
	}

	return c.JSON(fiber.Map{
		"translations": translations,
	})
}
