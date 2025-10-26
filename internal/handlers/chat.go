package handlers

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/developia-II/language-translator-backend/internal/database"
	"github.com/developia-II/language-translator-backend/internal/models"
	"github.com/developia-II/language-translator-backend/internal/services"
	"github.com/developia-II/language-translator-backend/utils"
)

var (
	groqOnce    sync.Once
	groqService *services.GroqService
)

func Chat(c *fiber.Ctx) error {

	groqOnce.Do(func() { groqService = services.NewGroqService() })
	var req models.ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := utils.Validate.Struct(req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	userID := c.Locals("userId").(string)
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	// Get or create conversation
	var conversation models.Conversation
	conversationCollection := database.GetCollection("conversations")

	if req.ConversationID == "" {
		// Create new conversation
		title := req.Message
		if len(title) > 50 {
			title = title[:50]
		}
		conversation = models.Conversation{
			ID:        primitive.NewObjectID(),
			UserID:    userObjID,
			Title:     title,
			Messages:  []models.Message{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		_, err := conversationCollection.InsertOne(context.Background(), conversation)
		if err != nil {
			return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create conversation")
		}
	} else {
		// Get existing conversation
		convObjID, _ := primitive.ObjectIDFromHex(req.ConversationID)
		err := conversationCollection.FindOne(context.Background(), bson.M{"_id": convObjID, "userId": userObjID}).Decode(&conversation)
		if err != nil {
			return utils.ErrorResponse(c, fiber.StatusNotFound, "Conversation not found")
		}
	}

	// Add user message to conversation
	userMessage := models.Message{
		ID:        primitive.NewObjectID(),
		Role:      "user",
		Content:   req.Message,
		Language:  req.Language,
		CreatedAt: time.Now(),
	}

	conversation.Messages = append(conversation.Messages, userMessage)

	// Build messages for Groq from conversation history
	msgs := services.BuildChatMessages(conversation.Messages, req.Language)
	aiResponse, err := groqService.Chat(context.Background(), msgs)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "AI service error: "+err.Error())
	}

	// Emergency detection and disclaimer handling
	finalContent := aiResponse
	if detectEmergency(req.Message) {
		finalContent = "Emergency warning: Your symptoms may be serious. Please seek immediate medical attention or contact local emergency services immediately.\n\n" + finalContent
	}
	finalContent = finalContent

	// Add AI response to conversation
	assistantMessage := models.Message{
		ID:        primitive.NewObjectID(),
		Role:      "assistant",
		Content:   finalContent,
		Language:  req.Language,
		CreatedAt: time.Now(),
	}

	conversation.Messages = append(conversation.Messages, assistantMessage)
	conversation.UpdatedAt = time.Now()

	// Save updated conversation
	_, err = conversationCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": conversation.ID},
		bson.M{"$set": conversation},
	)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to save conversation")
	}

	return c.JSON(models.ChatResponse{
		ConversationID: conversation.ID.Hex(),
		Message:        assistantMessage,
		Conversation:   &conversation,
	})
}

func GetConversations(c *fiber.Ctx) error {
	userID := c.Locals("userId").(string)
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	collection := database.GetCollection("conversations")
	cursor, err := collection.Find(context.Background(), bson.M{"userId": userObjID})
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to fetch conversations")
	}
	defer cursor.Close(context.Background())

	var conversations []models.Conversation
	if err := cursor.All(context.Background(), &conversations); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to decode conversations")
	}

	return c.JSON(fiber.Map{
		"conversations": conversations,
	})
}

func GetConversation(c *fiber.Ctx) error {
	conversationID := c.Params("id")
	convObjID, err := primitive.ObjectIDFromHex(conversationID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid conversation ID")
	}

	userID := c.Locals("userId").(string)
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	collection := database.GetCollection("conversations")
	var conversation models.Conversation
	err = collection.FindOne(context.Background(), bson.M{"_id": convObjID, "userId": userObjID}).Decode(&conversation)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "Conversation not found")
	}

	return c.JSON(fiber.Map{
		"conversation": conversation,
	})
}

// detectEmergency performs a simple keyword check for emergency symptoms.
func detectEmergency(text string) bool {
	lower := strings.ToLower(text)
	keywords := []string{
		"chest pain",
		"difficulty breathing",
		"shortness of breath",
		"severe bleeding",
		"unconscious",
		"fainting",
		"stroke",
		"heart attack",
		"suicidal",
		"overdose",
	}
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}
