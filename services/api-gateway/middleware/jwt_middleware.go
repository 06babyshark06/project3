package middlewares

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/06babyshark06/JQKStudy/services/api-gateway/converters"
	grpcclients "github.com/06babyshark06/JQKStudy/services/api-gateway/grpc_clients"
	"github.com/06babyshark06/JQKStudy/services/api-gateway/handlers"
	userpb "github.com/06babyshark06/JQKStudy/shared/proto/user"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

var identityKey = "user_id"

func NewJWTMiddleware(userClient *grpcclients.UserServiceClient) (*jwt.GinJWTMiddleware, error) {
	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "user zone",
		Key:         []byte("supersecretkey"),
		Timeout:     time.Minute * 15,
		MaxRefresh:  time.Hour * 24 * 7,
		IdentityKey: identityKey,

		Authenticator: func(c *gin.Context) (any, error) {
			var loginVals converters.LoginRequest
			if err := c.ShouldBindJSON(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}

			if handlers.Validate.Struct(loginVals) != nil {
				return "", jwt.ErrMissingLoginValues
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := userClient.Client.Login(ctx, &userpb.LoginRequest{
				Email:    loginVals.Email,
				Password: loginVals.Password,
			})
			if err != nil {
				log.Printf("❌ Login gRPC failed: %v", err)
				return nil, jwt.ErrFailedAuthentication
			}

			return map[string]any{
				"email":   resp.Email,
				"user_id": resp.Id,
				"role":    resp.Role,
			}, nil
		},

		// --- Khi sinh token ---
		PayloadFunc: func(data any) jwt.MapClaims {
			if v, ok := data.(map[string]any); ok {
				return jwt.MapClaims{
					"email":   v["email"],
					"user_id": v["user_id"],
					"role":    v["role"],
				}
			}
			return jwt.MapClaims{}
		},

		IdentityHandler: func(c *gin.Context) any {
			claims := jwt.ExtractClaims(c)

			c.Set("user_id", claims["user_id"])
			c.Set("email", claims["email"])
			c.Set("role", claims["role"])
			
			return map[string]any{
				"user_id": claims["user_id"],
				"email":   claims["email"],
				"role":    claims["role"],
			}
		},

		Authorizator: func(data any, c *gin.Context) bool {
			return true
		},

		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{"error": message})
		},

		TokenLookup:   "header: Authorization, query: token, cookie: jwt",
		TokenHeadName: "Bearer",
		TimeFunc:      time.Now,
	})
	if err != nil {
		return nil, err
	}

	return authMiddleware, nil
}

func Authorize(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Không tìm thấy vai trò (role) trong token"})
			return
		}

		userRole, ok := roleVal.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Vai trò (role) trong token không hợp lệ"})
			return
		}

		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Bạn không có quyền truy cập tài nguyên này"})
	}
}
