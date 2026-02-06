package logger

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log is the global logger instance
	Log *zap.Logger

	// Sensitive field patterns for redaction
	sensitivePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*\S+`),
		regexp.MustCompile(`(?i)(token|jwt|bearer)\s*[=:]\s*\S+`),
		regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[=:]\s*\S+`),
		regexp.MustCompile(`(?i)(secret|private[_-]?key)\s*[=:]\s*\S+`),
		regexp.MustCompile(`(?i)(authorization)\s*:\s*.*`),
	}

	// Sensitive field names
	sensitiveFields = map[string]bool{
		"password":    true,
		"passwd":      true,
		"pwd":         true,
		"token":       true,
		"jwt":         true,
		"bearer":      true,
		"api_key":     true,
		"apikey":      true,
		"secret":      true,
		"private_key": true,
		"privatekey":  true,
		"auth":        true,
		"authorization": true,
	}
)

// Config holds logger configuration
type Config struct {
	Level      string
	Format     string
	OutputPath string
}

// Initialize sets up the global logger
func Initialize(cfg Config) error {
	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("invalid log level %s: %w", cfg.Level, err)
	}

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if cfg.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Configure output
	var writeSyncer zapcore.WriteSyncer
	if cfg.OutputPath == "stdout" || cfg.OutputPath == "" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		writeSyncer = zapcore.AddSync(file)
	}

	// Create core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// Create logger with caller information
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return nil
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

// With creates a child logger with additional fields
func With(fields ...zap.Field) *zap.Logger {
	return Log.With(fields...)
}

// Sync flushes any buffered log entries
func Sync() error {
	return Log.Sync()
}

// RedactSensitiveData redacts sensitive information from a string
func RedactSensitiveData(data string) string {
	redacted := data
	for _, pattern := range sensitivePatterns {
		redacted = pattern.ReplaceAllStringFunc(redacted, func(match string) string {
			parts := strings.SplitN(match, "=", 2)
			if len(parts) == 2 {
				return parts[0] + "=[REDACTED]"
			}
			parts = strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				return parts[0] + ": [REDACTED]"
			}
			return "[REDACTED]"
		})
	}
	return redacted
}

// IsSensitiveField checks if a field name is sensitive
func IsSensitiveField(fieldName string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(fieldName, "-", "_"))
	return sensitiveFields[normalized]
}

// RedactField creates a redacted zap field for sensitive data
func RedactField(key string, value interface{}) zap.Field {
	if IsSensitiveField(key) {
		return zap.String(key, "[REDACTED]")
	}
	return zap.Any(key, value)
}

// SafeString creates a string field with automatic redaction
func SafeString(key, value string) zap.Field {
	if IsSensitiveField(key) {
		return zap.String(key, "[REDACTED]")
	}
	return zap.String(key, RedactSensitiveData(value))
}

// ErrorWithStack logs an error with stack trace
func ErrorWithStack(msg string, err error, fields ...zap.Field) {
	allFields := append(fields, zap.Error(err), zap.Stack("stack"))
	Log.Error(msg, allFields...)
}

// CriticalError logs a critical error that requires immediate attention
func CriticalError(msg string, err error, fields ...zap.Field) {
	allFields := append(fields, zap.Error(err), zap.Stack("stack"), zap.Bool("critical", true))
	Log.Error(msg, allFields...)
	// In production, this would trigger alerts to monitoring systems
}
