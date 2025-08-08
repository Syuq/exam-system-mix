package main

import (
	"context"
	"exam-system/config"
	"exam-system/handlers"
	"exam-system/middleware"
	"exam-system/models"
	"exam-system/services"
	"exam-system/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize logger
	logger := utils.InitLogger()

	// Initialize database
	db, err := models.InitDB()
	if err != nil {
		logger.Fatal("Failed to initialize database: ", err)
	}

	// Initialize Redis
	redisClient, err := utils.InitRedis()
	if err != nil {
		logger.Fatal("Failed to initialize Redis: ", err)
	}

	// Run migrations
	if err := models.RunMigrations(db); err != nil {
		logger.Fatal("Failed to run migrations: ", err)
	}

	// Initialize services
	authService := services.NewAuthService(db, redisClient, logger)
	userService := services.NewUserService(db, logger)
	questionService := services.NewQuestionService(db, logger)
	examService := services.NewExamService(db, redisClient, logger)
	resultService := services.NewResultService(db, logger)

	// Set Gin mode
	gin.SetMode(config.AppConfig.Server.GinMode)

	// Initialize Gin router
	router := gin.New()

	// Add middleware
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(middleware.CORSMiddleware())
	router.Use(gin.Recovery())

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, logger)
	userHandler := handlers.NewUserHandler(userService, logger)
	questionHandler := handlers.NewQuestionHandler(questionService, logger)
	examHandler := handlers.NewExamHandler(examService, logger)
	resultHandler := handlers.NewResultHandler(resultService, logger)

	// Setup routes
	setupRoutes(router, authHandler, userHandler, questionHandler, examHandler, resultHandler, redisClient, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + config.AppConfig.Server.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.WithFields(logrus.Fields{
			"port": config.AppConfig.Server.Port,
			"mode": config.AppConfig.Server.GinMode,
		}).Info("Starting server")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server: ", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown: ", err)
	}

	logger.Info("Server exited")
}

func setupRoutes(
	router *gin.Engine,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	questionHandler *handlers.QuestionHandler,
	examHandler *handlers.ExamHandler,
	resultHandler *handlers.ResultHandler,
	redisClient *utils.RedisClient,
	logger *logrus.Logger,
) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().Unix()})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")

	// Auth routes (with rate limiting)
	authGroup := v1.Group("/auth")
	authGroup.Use(middleware.RateLimitMiddleware(redisClient, config.AppConfig.RateLimit.LoginLimit, config.AppConfig.RateLimit.Window, "login"))
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.POST("/logout", middleware.AuthMiddleware(), authHandler.Logout)
	}

	// User routes
	userGroup := v1.Group("/users")
	userGroup.Use(middleware.AuthMiddleware())
	{
		userGroup.GET("/profile", userHandler.GetProfile)
		userGroup.PUT("/profile", userHandler.UpdateProfile)
		
		// Admin only routes
		adminUserGroup := userGroup.Group("")
		adminUserGroup.Use(middleware.AdminMiddleware())
		{
			adminUserGroup.GET("", userHandler.GetUsers)
			adminUserGroup.GET("/:id", userHandler.GetUser)
			adminUserGroup.PUT("/:id", userHandler.UpdateUser)
			adminUserGroup.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	// Question routes
	questionGroup := v1.Group("/questions")
	questionGroup.Use(middleware.AuthMiddleware())
	{
		questionGroup.GET("", questionHandler.GetQuestions)
		questionGroup.GET("/:id", questionHandler.GetQuestion)
		
		// Admin only routes
		adminQuestionGroup := questionGroup.Group("")
		adminQuestionGroup.Use(middleware.AdminMiddleware())
		{
			adminQuestionGroup.POST("", questionHandler.CreateQuestion)
			adminQuestionGroup.PUT("/:id", questionHandler.UpdateQuestion)
			adminQuestionGroup.DELETE("/:id", questionHandler.DeleteQuestion)
		}
	}

	// Exam routes
	examGroup := v1.Group("/exams")
	examGroup.Use(middleware.AuthMiddleware())
	{
		examGroup.GET("", examHandler.GetExams)
		examGroup.GET("/:id", examHandler.GetExam)
		examGroup.POST("/:id/start", examHandler.StartExam)
		examGroup.POST("/:id/submit", middleware.RateLimitMiddleware(redisClient, config.AppConfig.RateLimit.SubmitLimit, config.AppConfig.RateLimit.Window, "submit"), examHandler.SubmitExam)
		
		// Admin only routes
		adminExamGroup := examGroup.Group("")
		adminExamGroup.Use(middleware.AdminMiddleware())
		{
			adminExamGroup.POST("", examHandler.CreateExam)
			adminExamGroup.PUT("/:id", examHandler.UpdateExam)
			adminExamGroup.DELETE("/:id", examHandler.DeleteExam)
			adminExamGroup.POST("/:id/assign", examHandler.AssignExam)
		}
	}

	// Result routes
	resultGroup := v1.Group("/results")
	resultGroup.Use(middleware.AuthMiddleware())
	{
		resultGroup.GET("", resultHandler.GetResults)
		resultGroup.GET("/:id", resultHandler.GetResult)
		
		// Admin only routes
		adminResultGroup := resultGroup.Group("")
		adminResultGroup.Use(middleware.AdminMiddleware())
		{
			adminResultGroup.GET("/statistics", resultHandler.GetStatistics)
		}
	}

	// Admin routes
	adminGroup := v1.Group("/admin")
	adminGroup.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
	{
		adminGroup.POST("/seed", handlers.SeedData)
		adminGroup.GET("/logs", handlers.GetLogs)
	}
}

