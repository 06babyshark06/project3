package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	"github.com/06babyshark06/JQKStudy/services/api-gateway/handlers"
	middlewares "github.com/06babyshark06/JQKStudy/services/api-gateway/middleware"
	"github.com/06babyshark06/JQKStudy/services/api-gateway/redis"
	"github.com/06babyshark06/JQKStudy/shared/env"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	httpAddr := env.GetString("HTTP_ADDR", ":8081")
	redisAddr := env.GetString("REDIS_ADDR", "redis:6379")
	redis.InitRedis(redisAddr)

	userClient, err := grpcclients.NewUserServiceClient()
	if err != nil {
		log.Fatalf("failed to connect to user service: %v", err)
	}
	defer userClient.Close()

	examClient, err := grpcclients.NewExamServiceClient()
	if err != nil {
		log.Fatalf("failed to connect to exam service: %v", err)
	}
	defer examClient.Close()

	courseClient, err := grpcclients.NewCourseServiceClient()
	if err != nil {
		log.Fatalf("failed to connect to course service: %v", err)
	}
	defer courseClient.Close()

	jwtMiddleware, err := middlewares.NewJWTMiddleware(userClient)

	userHandler := handlers.NewUserHandler(userClient)
	authHandler := handlers.NewAuthHandler(userClient, jwtMiddleware)
	examHandler := handlers.NewExamHandler(examClient)
	courseHandler := handlers.NewCourseHandler(courseClient)
	statsHandler := handlers.NewStatsHandler(userClient, courseClient, examClient)

	// Tạo router
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://[::1]:3000"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
		// ExposeHeaders:    []string{"Authorization"},
		AllowCredentials: true,
	}))

	// Route test
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// === Public Routes (Không cần token) ===
		api.POST("/login", authHandler.Login)
		api.POST("/register", authHandler.Register)
		api.POST("/refresh", authHandler.Refresh)

		// Public GET (Ai cũng có thể xem)
		api.GET("/courses", courseHandler.GetCourses)
		api.GET("/topics", examHandler.GetTopics)
		api.GET("/exams", examHandler.GetExams)

		// Lưu ý: Get-Details nên nằm trong 'auth' vì nó cần userID
		// để check 'is_enrolled'

		// === Authenticated Routes (Yêu cầu token) ===
		auth := api.Group("/")
		auth.Use(jwtMiddleware.MiddlewareFunc())
		{
			// --- Routes cho TẤT CẢ user đã login ---
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/courses/:id", courseHandler.GetCourseDetails)
			auth.GET("/exams/:id", examHandler.GetExamDetails)

			// User tự quản lý
			userSelf := auth.Group("/users")
			{
				userSelf.GET("/me", userHandler.GetUserFromToken)
				userSelf.PUT("/me", userHandler.UpdateUserInfo)
				userSelf.PUT("/password", userHandler.UpdateUserPassword)
			}

			// Lấy chi tiết (cần userID từ token)
			// Lấy đề

			// --- Admin Only Routes ---
			adminOnly := auth.Group("/")
			adminOnly.Use(middlewares.Authorize("admin"))
			{
				adminUsers := adminOnly.Group("/users")
				{
					adminUsers.GET("", userHandler.GetAllUsers)
					adminUsers.GET("/:id", userHandler.GetUser)
					adminUsers.DELETE("/:id", userHandler.DeleteUser)
					adminUsers.PUT("/:id/role", userHandler.UpdateUserRole)
				}
				
				adminOnly.GET("/admin/stats", statsHandler.GetAdminStats)
				adminOnly.DELETE("/courses/:id", courseHandler.DeleteCourse)
			}

			// --- Instructor & Admin Routes ---
			instructorOnly := auth.Group("/")
			instructorOnly.Use(middlewares.Authorize("instructor", "admin"))
			{
				// Course Management
				instructorOnly.POST("/courses", courseHandler.CreateCourse)
				instructorOnly.PUT("/courses/:id", courseHandler.UpdateCourse)
				instructorOnly.PUT("/courses/:id/publish", courseHandler.PublishCourse)
				instructorOnly.GET("/instructor/courses", courseHandler.GetInstructorCourses)

				instructorOnly.POST("/sections", courseHandler.CreateSection)
				instructorOnly.PUT("/sections/:id", courseHandler.UpdateSection)
				instructorOnly.DELETE("/sections/:id", courseHandler.DeleteSection)

				instructorOnly.POST("/lessons", courseHandler.CreateLesson)
				instructorOnly.PUT("/lessons/:id", courseHandler.UpdateLesson)
				instructorOnly.DELETE("/lessons/:id", courseHandler.DeleteLesson)
				instructorOnly.POST("/lessons/upload-url", courseHandler.GetUploadURL)

				// Exam Management
				instructorOnly.POST("/topics", examHandler.CreateTopic)
				instructorOnly.POST("/questions", examHandler.CreateQuestion)
				instructorOnly.POST("/exams", examHandler.CreateExam)
				instructorOnly.PUT("/exams/:id", examHandler.UpdateExam)
				instructorOnly.PUT("/exams/:id/publish", examHandler.PublishExam)
				instructorOnly.GET("/instructor/exams", examHandler.GetInstructorExams)
				instructorOnly.PUT("/questions/:id", examHandler.UpdateQuestion)
				instructorOnly.DELETE("/questions/:id", examHandler.DeleteQuestion)
				instructorOnly.DELETE("/exams/:id", examHandler.DeleteExam)
			}

			// --- Student & Admin Routes ---
			studentOnly := auth.Group("/")
			studentOnly.Use(middlewares.Authorize("student", "admin"))
			{
				studentOnly.POST("/courses/enroll", courseHandler.EnrollCourse)
				studentOnly.GET("/my-courses", courseHandler.GetMyCourses)
				studentOnly.POST("/lessons/complete", courseHandler.MarkLessonCompleted)
				studentOnly.POST("/exams/submit", examHandler.SubmitExam)
				studentOnly.GET("/submissions/:id", examHandler.GetSubmission)
			}
		}
	}

	// HTTP server
	srv := &http.Server{
		Addr:    httpAddr,
		Handler: r,
	}

	serverErrors := make(chan error, 1)

	go func() {
		fmt.Printf("🚀 Server is running on %s\n", httpAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		fmt.Fprintf(os.Stderr, "❌ Server error: %v\n", err)

	case sig := <-shutdown:
		fmt.Printf("\n🛑 Caught signal: %v. Shutting down gracefully...\n", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Graceful shutdown failed: %v\n", err)
		} else {
			fmt.Println("✅ Server shut down cleanly.")
		}
	}
}
