package logger

import (
    "io"
    "os"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var (
    Log *zap.Logger
)

// Config 日志配置
type Config struct {
    Level      string
    Format     string
    OutputPath string
}

// Initialize 初始化日志器
func Initialize(config Config) error {
    var output io.Writer
    if config.OutputPath == "stdout" {
        output = os.Stdout
    } else {
        file, err := os.OpenFile(config.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            return err
        }
        output = file
    }

    level := zap.InfoLevel
    switch config.Level {
    case "debug":
        level = zap.DebugLevel
    case "info":
        level = zap.InfoLevel
    case "warn":
        level = zap.WarnLevel
    case "error":
        level = zap.ErrorLevel
    }

    var encoder zapcore.Encoder
    if config.Format == "json" {
        encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
    } else {
        encoder = zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
    }

    core := zapcore.NewCore(encoder, zapcore.AddSync(output), zap.NewAtomicLevelAt(level))
    Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

    return nil
}

// Sync 同步日志缓冲
func Sync() {
    if Log != nil {
        Log.Sync()
    }
}

// GetLogger 获取日志器
func GetLogger() *zap.Logger {
    if Log == nil {
        // 如果日志器未初始化，返回一个默认的
        Log, _ = zap.NewDevelopment()
    }
    return Log
}
