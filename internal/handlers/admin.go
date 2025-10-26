package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/developia-II/language-translator-backend/internal/database"
	"github.com/developia-II/language-translator-backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetAdminStats returns basic aggregate counts for the dashboard
func GetAdminStats(c *fiber.Ctx) error {
	ctx := context.Background()
	usersCol := database.GetCollection("users")
	translationsCol := database.GetCollection("translations")
	conversationsCol := database.GetCollection("conversations")
	feedbacksCol := database.GetCollection("feedbacks")

	usersCount, _ := usersCol.CountDocuments(ctx, bson.M{})
	translationsCount, _ := translationsCol.CountDocuments(ctx, bson.M{})
	conversationsCount, _ := conversationsCol.CountDocuments(ctx, bson.M{})
	feedbacksCount, _ := feedbacksCol.CountDocuments(ctx, bson.M{})

	return c.JSON(fiber.Map{
		"stats": fiber.Map{
			"totalUsers":         usersCount,
			"totalTranslations":  translationsCount,
			"totalConversations": conversationsCount,
			"totalFeedbacks":     feedbacksCount,
		},
	})
}

// GetAllUsers returns a list of users with basic public fields
func GetAllUsers(c *fiber.Ctx) error {
	ctx := context.Background()
	col := database.GetCollection("users")

	// Query params
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "20")
	q := c.Query("q", "")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	filter := bson.M{}
	if q != "" {
		// search by email or name (case-insensitive contains)
		filter = bson.M{"$or": []bson.M{
			{"email": bson.M{"$regex": q, "$options": "i"}},
			{"name": bson.M{"$regex": q, "$options": "i"}},
		}}
	}

	// Total count
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return utilsError(c)
	}

	// Pagination
	findOpts := options.Find().
		SetProjection(bson.M{"password": 0}).
		SetSort(bson.M{"createdAt": -1}).
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit))

	cursor, err := col.Find(ctx, filter, findOpts)
	if err != nil {
		return utilsError(c)
	}
	defer cursor.Close(ctx)

	var users []bson.M
	if err := cursor.All(ctx, &users); err != nil {
		return utilsError(c)
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"users":      users,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
	})
}

// GetAllFeedbacks returns all feedback documents
func GetAllFeedbacks(c *fiber.Ctx) error {
	ctx := context.Background()
	col := database.GetCollection("feedback")

	// Query params
	pageStr := c.Query("page", "1")
	limitStr := c.Query("limit", "20")
	fromStr := c.Query("from", "") // ISO8601
	toStr := c.Query("to", "")     // ISO8601

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	filter := bson.M{}
	// Date range filter
	if fromStr != "" || toStr != "" {
		createdAt := bson.M{}
		if fromStr != "" {
			if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
				createdAt["$gte"] = t
			}
		}
		if toStr != "" {
			if t, err := time.Parse(time.RFC3339, toStr); err == nil {
				createdAt["$lte"] = t
			}
		}
		if len(createdAt) > 0 {
			filter["createdAt"] = createdAt
		}
	}

	// Total count
	total, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return utilsError(c)
	}

	// Pagination options
	findOpts := options.Find().
		SetSort(bson.M{"createdAt": -1}).
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit))

	cursor, err := col.Find(ctx, filter, findOpts)
	if err != nil {
		return utilsError(c)
	}
	defer cursor.Close(ctx)

	var docs []models.Feedback
	if err := cursor.All(ctx, &docs); err != nil {
		return utilsError(c)
	}

	// Serialize ObjectIDs to hex strings
	feedbacks := make([]fiber.Map, 0, len(docs))
	for _, f := range docs {
		var tid, uid string
		if f.TranslationID != primitive.NilObjectID {
			tid = f.TranslationID.Hex()
		}
		if f.UserID != primitive.NilObjectID {
			uid = f.UserID.Hex()
		}
		feedbacks = append(feedbacks, fiber.Map{
			"id":            f.ID.Hex(),
			"translationId": tid,
			"userId":        uid,
			"rating":        f.Rating,
			"suggestedText": f.SuggestedText,
			"createdAt":     f.CreatedAt,
		})
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"feedbacks":  feedbacks,
		"page":       page,
		"limit":      limit,
		"total":      total,
		"totalPages": totalPages,
	})
}

// utilsError provides a generic internal error response for admin endpoints
func utilsError(c *fiber.Ctx) error {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "Internal server error",
	})
}
