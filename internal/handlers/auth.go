package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/developia-II/language-translator-backend/internal/database"
	"github.com/developia-II/language-translator-backend/internal/models"
	"github.com/developia-II/language-translator-backend/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func Signup(c *fiber.Ctx) error {
	var req models.SignupRequest
	if err := c.BodyParser(&req); err != nil {
		// Log detailed parse error and raw body for debugging
		log.Printf("Signup BodyParser error: %v; raw body: %s", err, string(c.Body()))
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := utils.Validate.Struct(req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	// Check if user exists
	collection := database.GetCollection("users")
	var existingUser models.User
	err := collection.FindOne(context.Background(), bson.M{"email": req.Email}).Decode(&existingUser)
	if err == nil {
		return utils.ErrorResponse(c, fiber.StatusConflict, "User already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to hash password")
	}

	// Create user
	user := models.User{
		ID:        primitive.NewObjectID(),
		Name:      req.Name,
		Email:     req.Email,
		Password:  string(hashedPassword),
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = collection.InsertOne(context.Background(), user)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create user")
	}

	// Generate JWT
	token, err := utils.GenerateJWT(user.ID.Hex(), user.Role)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"user": fiber.Map{
			"id":    user.ID.Hex(),
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
		"token": token,
	})
}

func Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		// Log detailed parse error and raw body for debugging
		log.Printf("Login BodyParser error: %v; raw body: %s", err, string(c.Body()))

		fmt.Println("ERROR", err)
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := utils.Validate.Struct(req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	// Find user
	collection := database.GetCollection("users")
	var user models.User
	err := collection.FindOne(context.Background(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Invalid credentials")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Invalid credentials")
	}

	// Generate JWT
	token, err := utils.GenerateJWT(user.ID.Hex(), user.Role)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":    user.ID.Hex(),
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
		"token": token,
	})
}

func AuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Missing authorization header")
	}

	tokenString := authHeader[7:] // Remove "Bearer " prefix

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)
	c.Locals("userId", claims["userId"].(string))
	if role, ok := claims["role"].(string); ok {
		c.Locals("role", role)
	}

	return c.Next()
}

// AdminMiddleware ensures the requester has role == "admin"
func AdminMiddleware(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if role != "admin" {
		return utils.ErrorResponse(c, fiber.StatusForbidden, "Admins only")
	}
	return c.Next()
}

// Me returns the authenticated user's profile
func Me(c *fiber.Ctx) error {
	userID, _ := c.Locals("userId").(string)
	if userID == "" {
		return utils.ErrorResponse(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	oid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid user id")
	}

	collection := database.GetCollection("users")
	var user models.User
	if err := collection.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&user); err != nil {
		return utils.ErrorResponse(c, fiber.StatusNotFound, "User not found")
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":    user.ID.Hex(),
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}
