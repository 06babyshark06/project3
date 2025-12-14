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
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://[::1]:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/login", authHandler.Login)
		api.POST("/register", authHandler.Register)
		api.POST("/refresh", authHandler.Refresh)
		api.GET("/courses", courseHandler.GetCourses)

		auth := api.Group("/")
		auth.Use(jwtMiddleware.MiddlewareFunc())
		{
			auth.POST("/logout", authHandler.Logout)

			auth.GET("/courses/:id", courseHandler.GetCourseDetails)
			auth.GET("/exams/:id", examHandler.GetExamDetails)
			auth.GET("/exams", examHandler.GetExams)

			auth.GET("/exam-sections", examHandler.GetSections)
			auth.GET("/topics", examHandler.GetTopics)

			userSelf := auth.Group("/users")
			{
				userSelf.GET("/me", userHandler.GetUserFromToken)
				userSelf.PUT("/me", userHandler.UpdateUserInfo)
				userSelf.PUT("/password", userHandler.UpdateUserPassword)
				userSelf.GET("/me/exam-stats", examHandler.GetMyExamStats)
			}

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

			instructorOnly := auth.Group("/")
			instructorOnly.Use(middlewares.Authorize("instructor", "admin"))
			{
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

				instructorOnly.POST("/topics", examHandler.CreateTopic)
				instructorOnly.POST("/exam-sections", examHandler.CreateSection)

				instructorOnly.GET("/questions", examHandler.GetQuestions)
				instructorOnly.GET("/questions/:id", examHandler.GetQuestion)
				instructorOnly.POST("/questions/import", examHandler.ImportQuestions)
				instructorOnly.POST("/questions/upload-url", examHandler.GetUploadURL)
				instructorOnly.POST("/questions", examHandler.CreateQuestion)
				instructorOnly.PUT("/questions/:id", examHandler.UpdateQuestion)
				instructorOnly.DELETE("/questions/:id", examHandler.DeleteQuestion)
				instructorOnly.GET("/questions/export", examHandler.ExportQuestions)

				instructorOnly.POST("/exams", examHandler.CreateExam)
				instructorOnly.POST("/exams/generate", examHandler.GenerateExam)
				instructorOnly.PUT("/exams/:id", examHandler.UpdateExam)
				instructorOnly.PUT("/exams/:id/publish", examHandler.PublishExam)
				instructorOnly.GET("/instructor/exams", examHandler.GetInstructorExams)
				instructorOnly.DELETE("/exams/:id", examHandler.DeleteExam)

				instructorOnly.PUT("/exams/access/approve", examHandler.ApproveAccess)
				instructorOnly.GET("/exams/:id/stats", examHandler.GetExamStats)
				instructorOnly.GET("/exams/:id/export", examHandler.ExportExamResults)
				instructorOnly.GET("/exams/:id/violations", examHandler.GetExamViolations)
			}

			studentOnly := auth.Group("/")
			studentOnly.Use(middlewares.Authorize("student", "admin", "instructor"))
			{
				studentOnly.POST("/courses/enroll", courseHandler.EnrollCourse)
				studentOnly.GET("/my-courses", courseHandler.GetMyCourses)
				studentOnly.POST("/lessons/complete", courseHandler.MarkLessonCompleted)

				studentOnly.POST("/exams/submit", examHandler.SubmitExam)
				studentOnly.GET("/submissions/:id", examHandler.GetSubmission)
				studentOnly.POST("/exams/access/request", examHandler.RequestAccess)
				studentOnly.GET("/exams/access/check", examHandler.CheckAccess)
				studentOnly.POST("/exams/save-answer", examHandler.SaveAnswer)
				studentOnly.POST("/exams/log-violation", examHandler.LogViolation)
				studentOnly.POST("/exams/:id/start", examHandler.StartExam)
			}
		}
	}

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
