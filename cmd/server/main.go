package main

import (
	"log"
	"os"
	"time"

	"github.com/developia-II/language-translator-backend/internal/database"
	"github.com/developia-II/language-translator-backend/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Disconnect()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: handlers.ErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("FRONTEND_URL"),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE",
	}))
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
	}))

	// Routes
	api := app.Group("/api/v1")

	// Auth routes
	auth := api.Group("/auth")
	auth.Post("/signup", handlers.Signup)
	auth.Post("/login", handlers.Login)
	auth.Get("/me", handlers.AuthMiddleware, handlers.Me)

	// Protected routes
	api.Use(handlers.AuthMiddleware)

	// Translation routes
	api.Post("/translate", handlers.Translate)
	api.Get("/translations", handlers.GetTranslations)

	// TTS route
	api.Post("/tts", handlers.TTS)

	// Feedback routes
	api.Post("/feedback", handlers.SubmitFeedback)
	api.Get("/feedback/:translationId", handlers.GetFeedback)

	// Chat routes (protected)
	api.Use(handlers.AuthMiddleware)
	api.Post("/chat", handlers.Chat)
	api.Get("/conversations", handlers.GetConversations)
	api.Get("/conversations/:id", handlers.GetConversation)

	// Admin routes (protected by Auth + Admin middleware)
	admin := api.Group("/admin")
	admin.Use(handlers.AdminMiddleware)
	admin.Get("/stats", handlers.GetAdminStats)
	admin.Get("/feedbacks", handlers.GetAllFeedbacks)
	admin.Get("/users", handlers.GetAllUsers)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("GROQ_API_KEY present: %v", os.Getenv("GROQ_API_KEY") != "")
	log.Printf("GROQ_MODEL: %s", os.Getenv("GROQ_MODEL"))

	log.Printf("Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
