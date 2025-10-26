package handlers

import (
	"context"
	"strconv"
	"strings"
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

	// active users in last 7 days based on translations activity
	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour)
	activePipeline := []bson.M{
		{"$match": bson.M{"createdAt": bson.M{"$gte": sevenDaysAgo}}},
		{"$group": bson.M{"_id": "$userId"}},
		{"$count": "count"},
	}

	var activeResult []bson.M
	cur, err := translationsCol.Aggregate(ctx, activePipeline)
	if err == nil {
		_ = cur.All(ctx, &activeResult)
	}
	activeUsers := int64(0)
	if len(activeResult) > 0 {
		if v, ok := activeResult[0]["count"].(int32); ok {
			activeUsers = int64(v)
		} else if v, ok := activeResult[0]["count"].(int64); ok {
			activeUsers = v
		}
	}

	// average feedback rating
	avgPipeline := []bson.M{
		{"$group": bson.M{"_id": nil, "avg": bson.M{"$avg": "$rating"}}},
	}
	var avgResult []bson.M
	cur2, err := feedbacksCol.Aggregate(ctx, avgPipeline)
	if err == nil {
		_ = cur2.All(ctx, &avgResult)
	}
	avgFeedback := 0.0
	if len(avgResult) > 0 {
		switch v := avgResult[0]["avg"].(type) {
		case float64:
			avgFeedback = v
		case float32:
			avgFeedback = float64(v)
		}
	}

	return c.JSON(fiber.Map{
		"stats": fiber.Map{
			"totalUsers":         usersCount,
			"activeUsers":        activeUsers,
			"totalTranslations":  translationsCount,
			"totalConversations": conversationsCount,
			"totalFeedbacks":     feedbacksCount,
			"avgFeedbackRating":  avgFeedback,
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
	col := database.GetCollection("feedbacks")

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

// GetUserGrowth returns daily user signups within a range (default 30d)
func GetUserGrowth(c *fiber.Ctx) error {
	ctx := context.Background()
	col := database.GetCollection("users")
	rangeStr := c.Query("range", "30d")
	days := 30
	if strings.HasSuffix(rangeStr, "d") {
		if v, err := strconv.Atoi(strings.TrimSuffix(rangeStr, "d")); err == nil && v > 0 && v <= 180 {
			days = v
		}
	}
	from := time.Now().Add(time.Duration(-days) * 24 * time.Hour)
	pipeline := []bson.M{
		{"$match": bson.M{"createdAt": bson.M{"$gte": from}}},
		{"$group": bson.M{
			"_id": bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$createdAt"}},
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"_id": 1}},
	}
	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return utilsError(c)
	}
	defer cursor.Close(ctx)
	type pt struct{ Date string `json:"date"`; Count int `json:"count"` }
	var out []pt
	for cursor.Next(ctx) {
		var m bson.M
		if err := cursor.Decode(&m); err != nil { continue }
		out = append(out, pt{Date: m["_id"].(string), Count: int(m["count"].(int32))})
	}
	return c.JSON(fiber.Map{"series": out})
}

// GetTranslationVolume returns daily number of translations within a range (default 30d)
func GetTranslationVolume(c *fiber.Ctx) error {
	ctx := context.Background()
	col := database.GetCollection("translations")
	rangeStr := c.Query("range", "30d")
	days := 30
	if strings.HasSuffix(rangeStr, "d") {
		if v, err := strconv.Atoi(strings.TrimSuffix(rangeStr, "d")); err == nil && v > 0 && v <= 180 {
			days = v
		}
	}
	from := time.Now().Add(time.Duration(-days) * 24 * time.Hour)
	pipeline := []bson.M{
		{"$match": bson.M{"createdAt": bson.M{"$gte": from}}},
		{"$group": bson.M{
			"_id": bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$createdAt"}},
			"count": bson.M{"$sum": 1},
		}},
		{"$sort": bson.M{"_id": 1}},
	}
	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return utilsError(c)
	}
	defer cursor.Close(ctx)
	type pt struct{ Date string `json:"date"`; Count int `json:"count"` }
	var out []pt
	for cursor.Next(ctx) {
		var m bson.M
		if err := cursor.Decode(&m); err != nil { continue }
		out = append(out, pt{Date: m["_id"].(string), Count: int(m["count"].(int32))})
	}
	return c.JSON(fiber.Map{"series": out})
}

// GetFeedbackDistribution returns counts per rating over a range (default 30d)
func GetFeedbackDistribution(c *fiber.Ctx) error {
	ctx := context.Background()
	col := database.GetCollection("feedbacks")
	rangeStr := c.Query("range", "30d")
	days := 30
	if strings.HasSuffix(rangeStr, "d") {
		if v, err := strconv.Atoi(strings.TrimSuffix(rangeStr, "d")); err == nil && v > 0 && v <= 180 {
			days = v
		}
	}
	from := time.Now().Add(time.Duration(-days) * 24 * time.Hour)
	pipeline := []bson.M{
		{"$match": bson.M{"createdAt": bson.M{"$gte": from}}},
		{"$group": bson.M{ "_id": "$rating", "count": bson.M{"$sum": 1}}},
		{"$sort": bson.M{"_id": 1}},
	}
	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return utilsError(c)
	}
	defer cursor.Close(ctx)
	type item struct{ Rating int `json:"rating"`; Count int `json:"count"` }
	var dist []item
	for cursor.Next(ctx) {
		var m bson.M
		if err := cursor.Decode(&m); err != nil { continue }
		rating := 0
		switch v := m["_id"].(type) {
		case int32:
			rating = int(v)
		case int64:
			rating = int(v)
		}
		dist = append(dist, item{Rating: rating, Count: int(m["count"].(int32))})
	}
	return c.JSON(fiber.Map{"distribution": dist})
}

// GetTranslationByLanguage returns counts of translations by target language within a range (default 30d)
func GetTranslationByLanguage(c *fiber.Ctx) error {
    ctx := context.Background()
    col := database.GetCollection("translations")
    rangeStr := c.Query("range", "30d")
    days := 30
    if strings.HasSuffix(rangeStr, "d") {
        if v, err := strconv.Atoi(strings.TrimSuffix(rangeStr, "d")); err == nil && v > 0 && v <= 180 {
            days = v
        }
    }
    from := time.Now().Add(time.Duration(-days) * 24 * time.Hour)
    pipeline := []bson.M{
        {"$match": bson.M{"createdAt": bson.M{"$gte": from}}},
        {"$group": bson.M{
            "_id": "$targetLang",
            "count": bson.M{"$sum": 1},
        }},
        {"$sort": bson.M{"count": -1}},
    }
    cursor, err := col.Aggregate(ctx, pipeline)
    if err != nil {
        return utilsError(c)
    }
    defer cursor.Close(ctx)
    type item struct{ Language string `json:"language"`; Count int `json:"count"` }
    var out []item
    for cursor.Next(ctx) {
        var m bson.M
        if err := cursor.Decode(&m); err != nil { continue }
        lang := "Unknown"
        if s, ok := m["_id"].(string); ok && s != "" {
            lang = s
        }
        cnt := 0
        switch v := m["count"].(type) {
        case int32:
            cnt = int(v)
        case int64:
            cnt = int(v)
        }
        out = append(out, item{Language: lang, Count: cnt})
    }
    return c.JSON(fiber.Map{"languages": out})
}
