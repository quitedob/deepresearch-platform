package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGenerateQuestionsAuthorizesSessionBeforeGeneration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for isolated PostgreSQL tests")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	connection, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	var databaseName string
	if err := database.Raw("SELECT current_database()").Scan(&databaseName).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(databaseName, "_test") {
		t.Fatalf("refusing to modify non-test database %q", databaseName)
	}
	if err := database.AutoMigrate(&model.AIQuestionSession{}); err != nil {
		t.Fatal(err)
	}
	ownerID := uuid.NewString()
	session := &model.AIQuestionSession{UserID: ownerID, Title: "Ownership regression", Provider: "test", Model: "test"}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Unscoped().Delete(session) })
	api := NewAIQuestionAPIFull(nil, dao.NewAIQuestionDAO(database))
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name       string
		userID     string
		sessionID  string
		wantStatus int
	}{
		{name: "other user is forbidden", userID: uuid.NewString(), sessionID: session.ID, wantStatus: http.StatusForbidden},
		{name: "unknown session is not found", userID: ownerID, sessionID: uuid.NewString(), wantStatus: http.StatusNotFound},
		{name: "unauthenticated request is rejected", sessionID: session.ID, wantStatus: http.StatusUnauthorized},
		{name: "owner reaches generation", userID: ownerID, sessionID: session.ID, wantStatus: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			engine := gin.New()
			engine.POST("/generate", func(ctx *gin.Context) {
				if test.userID != "" {
					ctx.Set("user_id", test.userID)
				}
				api.GenerateQuestions(ctx)
			})
			request := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewBufferString(`{"prompt":"Generate one question","session_id":"`+test.sessionID+`"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
		})
	}
}
