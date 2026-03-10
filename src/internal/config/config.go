// Deprecated: This package is superseded by github.com/ai-research-platform/internal/infrastructure/config.
// All new code should import from infrastructure/config. This package will be removed in a future version.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Research ResearchConfig `mapstructure:"research"`
	Security SecurityConfig `mapstructure:"security"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
	Port         int `mapstructure:"port"`
	Env          string `mapstructure:"env"`
	ReadTimeout  int `mapstructure:"read_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"`
	DBName         string `mapstructure:"db_name"`
	SSLMode        string `mapstructure:"ssl_mode"`
	MaxConnections int    `mapstructure:"max_connections"`
	IdleConnections int   `mapstructure:"idle_connections"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	TTL      int    `mapstructure:"ttl"`
}

type LLMConfig struct {
	DefaultProvider string                    `mapstructure:"default_provider"`
	Timeout         int                       `mapstructure:"timeout"`
	Retries         int                       `mapstructure:"retries"`
	Providers       map[string]ProviderConfig `mapstructure:"providers"`
}

type ProviderConfig struct {
	APIKey       string   `mapstructure:"api_key"`
	BaseURL      string   `mapstructure:"base_url"`
	Models       []string `mapstructure:"models"`
	DefaultModel string   `mapstructure:"default_model"`
	Temperature  float64  `mapstructure:"temperature"`
	MaxTokens    int      `mapstructure:"max_tokens"`
}

type ResearchConfig struct {
	MaxIterations  int  `mapstructure:"max_iterations"`
	SessionTimeout int  `mapstructure:"session_timeout"`
	MaxSources     int  `mapstructure:"max_sources"`
	EnableCache    bool `mapstructure:"enable_cache"`
	WorkerPoolSize int  `mapstructure:"worker_pool_size"`
}

type SecurityConfig struct {
	JWTSecret      string         `mapstructure:"jwt_secret"`
	JWTExpiration  int            `mapstructure:"jwt_expiration"`
	BcryptCost     int            `mapstructure:"bcrypt_cost"`
	CORS           CORSConfig     `mapstructure:"cors"`
	RateLimit      RateLimitConfig `mapstructure:"rate_limit"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods"`
}

type RateLimitConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	RequestsPerMin  int  `mapstructure:"requests_per_min"`
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Try environment-specific config first
		env := getEnv()
		v.SetConfigName(fmt.Sprintf("config.%s", env))
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		
		// Try to read environment-specific config
		if err := v.ReadInConfig(); err != nil {
			// Fall back to default config.yaml
			v.SetConfigName("config")
		}
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Enable environment variable override
	// Environment variables take precedence over config file values
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("APP")
	
	// Bind specific keys to enable environment variable override
	bindEnvKeys(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// bindEnvKeys binds all configuration keys to environment variables
// This allows environment variables to override config file values
func bindEnvKeys(v *viper.Viper) {
	keys := []string{
		// Server keys
		"server.port",
		"server.env",
		"server.read_timeout",
		"server.write_timeout",
		// Database keys
		"database.host",
		"database.port",
		"database.user",
		"database.password",
		"database.db_name",
		"database.ssl_mode",
		"database.max_connections",
		"database.idle_connections",
		// Redis keys
		"redis.host",
		"redis.port",
		"redis.password",
		"redis.db",
		"redis.ttl",
		// LLM keys
		"llm.default_provider",
		"llm.timeout",
		"llm.retries",
		// Research keys
		"research.max_iterations",
		"research.session_timeout",
		"research.max_sources",
		"research.enable_cache",
		"research.worker_pool_size",
		// Security keys
		"security.jwt_secret",
		"security.jwt_expiration",
		"security.bcrypt_cost",
		"security.rate_limit.enabled",
		"security.rate_limit.requests_per_min",
		// Logging keys
		"logging.level",
		"logging.format",
		"logging.output_path",
	}
	
	for _, key := range keys {
		v.BindEnv(key)
	}
}

// getEnv returns the current environment (dev, staging, prod)
func getEnv() string {
	env := strings.ToLower(strings.TrimSpace(viper.GetString("APP_ENV")))
	if env == "" {
		env = "development"
	}
	return env
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Server validation
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Database validation
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	// LLM validation
	if c.LLM.DefaultProvider == "" {
		return fmt.Errorf("default LLM provider is required")
	}
	if _, exists := c.LLM.Providers[c.LLM.DefaultProvider]; !exists {
		return fmt.Errorf("default provider %s not found in providers", c.LLM.DefaultProvider)
	}

	// Security validation
	if c.Security.JWTSecret == "" || c.Security.JWTSecret == "change-this-secret-in-production" {
		if c.Server.Env == "production" {
			return fmt.Errorf("JWT secret must be set in production")
		}
	}
	if c.Security.BcryptCost < 10 || c.Security.BcryptCost > 31 {
		return fmt.Errorf("bcrypt cost must be between 10 and 31")
	}

	return nil
}

// GetReadTimeout returns the server read timeout as a duration
func (c *Config) GetReadTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeout) * time.Second
}

// GetWriteTimeout returns the server write timeout as a duration
func (c *Config) GetWriteTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeout) * time.Second
}

// GetJWTExpiration returns the JWT expiration as a duration
func (c *Config) GetJWTExpiration() time.Duration {
	return time.Duration(c.Security.JWTExpiration) * time.Second
}

// GetLLMTimeout returns the LLM timeout as a duration
func (c *Config) GetLLMTimeout() time.Duration {
	return time.Duration(c.LLM.Timeout) * time.Second
}

// GetResearchTimeout returns the research session timeout as a duration
func (c *Config) GetResearchTimeout() time.Duration {
	return time.Duration(c.Research.SessionTimeout) * time.Second
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns the Redis address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}
