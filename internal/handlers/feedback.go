package handlers

import (
	"context"
	"time"

	"github.com/developia-II/language-translator-backend/internal/database"
	"github.com/developia-II/language-translator-backend/internal/models"
	"github.com/developia-II/language-translator-backend/utils"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func SubmitFeedback(c *fiber.Ctx) error {
	var req models.FeedbackRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := utils.Validate.Struct(req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	userID := c.Locals("userId").(string)
	userObjID, _ := primitive.ObjectIDFromHex(userID)
	translationObjID, _ := primitive.ObjectIDFromHex(req.TranslationID)

	feedback := models.Feedback{
		ID:            primitive.NewObjectID(),
		TranslationID: translationObjID,
		UserID:        userObjID,
		Rating:        req.Rating,
		SuggestedText: req.SuggestedText,
		CreatedAt:     time.Now(),
	}

	collection := database.GetCollection("feedback")
	_, err := collection.InsertOne(context.Background(), feedback)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to save feedback")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"feedback": feedback,
	})
}

func GetFeedback(c *fiber.Ctx) error {
	translationID := c.Params("translationId")
	translationObjID, err := primitive.ObjectIDFromHex(translationID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid translation ID")
	}

	collection := database.GetCollection("feedback")

	var feedbacks []models.Feedback
	cursor, err := collection.Find(context.Background(), bson.M{"translationId": translationObjID})
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch feedback")
	}
	defer cursor.Close(context.Background())

	if err := cursor.All(context.Background(), &feedbacks); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to decode feedback")
	}

	return c.JSON(fiber.Map{
		"feedback": feedbacks,
	})
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}
