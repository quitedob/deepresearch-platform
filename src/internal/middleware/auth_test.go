package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ai-research-platform/internal/pkg/auth"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type testUserStatusChecker struct {
	user *model.User
	err  error
}

func (checker *testUserStatusChecker) FindByID(context.Context, string) (*model.User, error) {
	return checker.user, checker.err
}

func TestAuthWithJWTChecksCurrentAccountStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	token, err := jwtManager.GenerateToken("user-1", "stale@example.com", "stale")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		checker    *testUserStatusChecker
		wantStatus int
	}{
		{
			name: "active account uses database identity",
			checker: &testUserStatusChecker{user: &model.User{
				ID: "user-1", Email: "current@example.com", Username: "current", Status: "active",
			}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "banned account",
			checker:    &testUserStatusChecker{user: &model.User{ID: "user-1", Status: "banned"}},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "missing account",
			checker:    &testUserStatusChecker{err: gorm.ErrRecordNotFound},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "lookup failure",
			checker:    &testUserStatusChecker{err: errors.New("database unavailable")},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	originalChecker := userStatusChecker
	defer func() { userStatusChecker = originalChecker }()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userStatusChecker = test.checker
			engine := gin.New()
			engine.Use(AuthWithJWT(jwtManager))
			engine.GET("/protected", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"email":    c.GetString("email"),
					"username": c.GetString("username"),
				})
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantStatus == http.StatusOK && response.Body.String() != "{\"email\":\"current@example.com\",\"username\":\"current\"}" {
				t.Fatalf("database identity not applied: %s", response.Body.String())
			}
		})
	}
}
