package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/06babyshark06/JQKStudy/services/api-gateway/converters"
	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	"github.com/06babyshark06/JQKStudy/services/api-gateway/redis"
	"github.com/06babyshark06/JQKStudy/shared/contracts"
	pb "github.com/06babyshark06/JQKStudy/shared/proto/user"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	jwtv4 "github.com/golang-jwt/jwt/v4"
)

type AuthHandler struct {
	AuthClient pb.UserServiceClient
	jwt        *jwt.GinJWTMiddleware
}

func NewAuthHandler(client *grpcclients.UserServiceClient, jwt *jwt.GinJWTMiddleware) *AuthHandler {
	return &AuthHandler{AuthClient: client.Client, jwt: jwt}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req converters.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, contracts.APIResponse{Error: &contracts.APIError{Code: "binding_error", Message: err.Error()}})
		return
	}

	if err := Validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, contracts.APIResponse{Error: &contracts.APIError{Code: "validation_error", Message: err.Error()}})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	protoReq := converters.ConvertRegisterJSONToProto(&req)

	resp, err := h.AuthClient.Register(ctx, protoReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, contracts.APIResponse{Error: &contracts.APIError{Code: "internal_error", Message: err.Error()}})
		return
	}

	c.JSON(http.StatusCreated, contracts.APIResponse{Data: resp})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req converters.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := h.AuthClient.Login(ctx, &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	accessToken, _, err := h.jwt.TokenGenerator(map[string]any{
		"user_id":   resp.Id,
		"email":     resp.Email,
		"role":      resp.Role,
		"full_name": resp.FullName,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshClaims := jwtv4.MapClaims{
		"user_id":   resp.Id,
		"email":     resp.Email,
		"role":      resp.Role,
		"full_name": resp.FullName,
		"exp":       time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	refreshToken := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString([]byte("supersecretkey"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// ===== Lưu refresh token vào Redis =====
	if err := redis.SetRefreshToken(fmt.Sprint(resp.Id), refreshTokenStr, 7*24*time.Hour); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refresh token"})
		return
	}

	c.SetCookie(
		"refresh_token",
		refreshTokenStr,
		int(h.jwt.MaxRefresh.Seconds()),
		"/",
		"localhost",
		false,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"expires_in":   int(h.jwt.Timeout.Seconds()),
		"user": gin.H{
			"id":        resp.Id,
			"full_name": resp.FullName,
			"email":     resp.Email,
			"role":      resp.Role,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	claims := jwt.ExtractClaims(c)
	userIDValue, ok := claims["user_id"]
	if !ok {
		c.JSON(http.StatusUnauthorized, contracts.APIResponse{
			Error: &contracts.APIError{Code: "unauthorized", Message: "missing user_id in token"},
		})
		return
	}

	var userID string
	switch v := userIDValue.(type) {
	case float64:
		userID = fmt.Sprintf("%.0f", v)
	case string:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, contracts.APIResponse{
			Error: &contracts.APIError{Code: "internal_error", Message: "invalid user_id type"},
		})
		return
	}

	if err := redis.DeleteRefreshToken(userID); err != nil {
		c.JSON(http.StatusInternalServerError, contracts.APIResponse{
			Error: &contracts.APIError{Code: "internal_error", Message: err.Error()},
		})
		return
	}

	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/api/v1/refresh",
		"",
		false,
		true,
	)

	c.JSON(http.StatusOK, contracts.APIResponse{Data: gin.H{"message": "logout success"}})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshTokenStr, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing refresh token cookie"})
		return
	}

	token, err := h.jwt.ParseTokenString(refreshTokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	claims, ok := token.Claims.(jwtv4.MapClaims)
	if !ok || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}

	userID := fmt.Sprint(claims["user_id"])

	storedToken, err := redis.GetRefreshToken(userID)
	if err != nil || storedToken != refreshTokenStr {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired or invalid"})
		return
	}

	newToken, _, err := h.jwt.TokenGenerator(map[string]any{
		"user_id":   claims["user_id"],
		"email":     claims["email"],
		"role":      claims["role"],
		"full_name": claims["full_name"],
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": newToken,
	})
}
