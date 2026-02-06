package middleware

import (
    "net/http"
    "runtime/debug"

    "github.com/gin-gonic/gin"
    "github.com/ai-research-platform/internal/logger"
    "go.uber.org/zap"
)

// Recovery 鎭㈠涓棿浠?
func Recovery() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                // 璁板綍panic鍫嗘爤淇℃伅
                debug.PrintStack()

                // 浣跨敤缁撴瀯鍖栨棩蹇楄褰曢敊璇?
                logger.Error("panic recovered",
                    zap.Any("error", err),
                    zap.String("stack", string(debug.Stack())),
                    zap.String("path", c.Request.URL.Path),
                    zap.String("method", c.Request.Method),
                )

                c.JSON(http.StatusInternalServerError, gin.H{
                    "error": "internal server error",
                })
                c.Abort()
            }
        }()

        c.Next()
    }
}

// CustomRecovery 鑷畾涔夋仮澶嶄腑闂翠欢
func CustomRecovery(handler func(c *gin.Context, err interface{})) gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                handler(c, err)
                c.Abort()
            }
        }()

        c.Next()
    }
}
