package v1

import (
    "context"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/ai-research-platform/internal/cache"
    "github.com/ai-research-platform/internal/types/response"
    "gorm.io/gorm"
)

// HealthAPI 鍋ュ悍妫€鏌PI
type HealthAPI struct {
    db    *gorm.DB
    cache cache.Cache
}

// NewHealthAPI 鍒涘缓鍋ュ悍妫€鏌PI
func NewHealthAPI(db *gorm.DB, cacheManager cache.Cache) *HealthAPI {
    return &HealthAPI{
        db:    db,
        cache: cacheManager,
    }
}

// Health 鍩烘湰鍋ュ悍妫€鏌?
// @Summary 鍋ュ悍妫€鏌?
// @Description 妫€鏌ユ湇鍔″熀鏈仴搴风姸鍐?
// @Tags health
// @Produce json
// @Success 200 {object} response.HealthResponse
// @Router /health [get]
func (h *HealthAPI) Health(c *gin.Context) {
    c.JSON(200, response.HealthResponse{
        Status: "healthy",
    })
}

// Ready 灏辩华妫€鏌?
// @Summary 灏辩华妫€鏌?
// @Description 妫€鏌ユ湇鍔℃槸鍚﹀噯澶囧ソ鎺ユ敹娴侀噺
// @Tags health
// @Produce json
// @Success 200 {object} response.ReadinessResponse
// @Failure 503 {object} response.ReadinessResponse
// @Router /ready [get]
func (h *HealthAPI) Ready(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
    defer cancel()

    checks := make(map[string]string)
    allReady := true

    // 妫€鏌ユ暟鎹簱杩炴帴
    if h.db != nil {
        sqlDB, err := h.db.DB()
        if err != nil {
            checks["database"] = "error: " + err.Error()
            allReady = false
        } else if err := sqlDB.PingContext(ctx); err != nil {
            checks["database"] = "error: " + err.Error()
            allReady = false
        } else {
            checks["database"] = "ok"
        }
    } else {
        checks["database"] = "not configured"
    }

    // 妫€鏌ョ紦瀛樿繛鎺?
    if h.cache != nil {
        if err := h.cache.Set(ctx, "health_check", "ok", time.Second); err != nil {
            checks["cache"] = "error: " + err.Error()
            allReady = false
        } else {
            checks["cache"] = "ok"
        }
    } else {
        checks["cache"] = "not configured"
    }

    status := "ready"
    statusCode := 200
    if !allReady {
        status = "not ready"
        statusCode = 503
    }

    c.JSON(statusCode, response.ReadinessResponse{
        Status: status,
        Checks: checks,
    })
}
