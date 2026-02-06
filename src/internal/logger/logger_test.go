package logger

import (
	"testing"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "json format with info level",
			config: Config{
				Level:      "info",
				Format:     "json",
				OutputPath: "stdout",
			},
			wantErr: false,
		},
		{
			name: "console format with debug level",
			config: Config{
				Level:      "debug",
				Format:     "console",
				OutputPath: "stdout",
			},
			wantErr: false,
		},
		{
			name: "invalid log level",
			config: Config{
				Level:      "invalid",
				Format:     "json",
				OutputPath: "stdout",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoggerFunctions(t *testing.T) {
	// Initialize logger for testing
	err := Initialize(Config{
		Level:      "debug",
		Format:     "json",
		OutputPath: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Test that logging functions don't panic
	t.Run("Info", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Info() panicked: %v", r)
			}
		}()
		Info("test info message")
	})

	t.Run("Debug", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debug() panicked: %v", r)
			}
		}()
		Debug("test debug message")
	})

	t.Run("Warn", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warn() panicked: %v", r)
			}
		}()
		Warn("test warn message")
	})

	t.Run("Error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error() panicked: %v", r)
			}
		}()
		Error("test error message")
	})
}
