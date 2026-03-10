package config

import (
    "fmt"
    "os"
    "regexp"
    "strings"
    "time"
    
    "github.com/joho/godotenv"
    "github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
    Server    ServerConfig    `mapstructure:"server"`
    Database  DatabaseConfig  `mapstructure:"database"`
    Redis     RedisConfig     `mapstructure:"redis"`
    LLM       LLMConfig       `mapstructure:"llm"`
    Research  ResearchConfig  `mapstructure:"research"`
    Security  SecurityConfig  `mapstructure:"security"`
    Logging   LoggingConfig   `mapstructure:"logging"`
    Admin     AdminConfig     `mapstructure:"admin"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
    Port         int    `mapstructure:"port"`
    Env          string `mapstructure:"env"`
    ReadTimeout  int    `mapstructure:"read_timeout"`
    WriteTimeout int    `mapstructure:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
    Host            string `mapstructure:"host"`
    Port            int    `mapstructure:"port"`
    User            string `mapstructure:"user"`
    Password        string `mapstructure:"password"`
    DBName          string `mapstructure:"db_name"`
    SSLMode         string `mapstructure:"ssl_mode"`
    MaxConnections  int    `mapstructure:"max_connections"`
    IdleConnections int    `mapstructure:"idle_connections"`
}

// RedisConfig Redis配置
type RedisConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
    TTL      int    `mapstructure:"ttl"`
}

// LLMConfig LLM配置
type LLMConfig struct {
    DefaultProvider string                    `mapstructure:"default_provider"`
    Timeout         int                       `mapstructure:"timeout"`
    Retries         int                       `mapstructure:"retries"`
    Providers       map[string]ProviderConfig `mapstructure:"providers"`
}

// ProviderConfig LLM提供商配置
type ProviderConfig struct {
    APIKey  string   `mapstructure:"api_key"`
    BaseURL string   `mapstructure:"base_url"`
    Models  []string `mapstructure:"models"`
}

// ResearchConfig 研究配置
type ResearchConfig struct {
    MaxIterations   int  `mapstructure:"max_iterations"`
    SessionTimeout  int  `mapstructure:"session_timeout"`
    MaxSources      int  `mapstructure:"max_sources"`
    EnableCache     bool `mapstructure:"enable_cache"`
    WorkerPoolSize  int  `mapstructure:"worker_pool_size"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
    JWTSecret     string      `mapstructure:"jwt_secret"`
    JWTExpiration int         `mapstructure:"jwt_expiration"`
    BcryptCost    int         `mapstructure:"bcrypt_cost"`
    CORS          CORSConfig  `mapstructure:"cors"`
    RateLimit     RateLimitConfig `mapstructure:"rate_limit"`
}

// AdminConfig 管理员配置
type AdminConfig struct {
    Email    string `mapstructure:"email"`
    Username string `mapstructure:"username"`
    Password string `mapstructure:"password"`
}

// CORSConfig CORS配置
type CORSConfig struct {
    AllowOrigins []string `mapstructure:"allow_origins"`
    AllowMethods []string `mapstructure:"allow_methods"`
    AllowHeaders []string `mapstructure:"allow_headers"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
    Enabled        bool `mapstructure:"enabled"`
    RequestsPerMin int  `mapstructure:"requests_per_min"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
    Level      string `mapstructure:"level"`
    Format     string `mapstructure:"format"`
    OutputPath string `mapstructure:"output_path"`
}

// Load 加载配置
func Load(configPath string) (*Config, error) {
    // 加载 .env 文件（如果存在）
    godotenv.Load() // 忽略错误

    // 读取 YAML 配置文件
    viper.SetConfigFile(configPath)
    viper.SetConfigType("yaml")
    
    // 设置环境变量自动读取
    viper.AutomaticEnv()

    // 读取配置文件
    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    // 展开 ${ENV_VAR:default} 模式的环境变量引用
    expandEnvVars()

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, err
    }

    // 从环境变量覆盖 API 密钥
    if deepseekKey := os.Getenv("DEEPSEEK_API_KEY"); deepseekKey != "" && deepseekKey != "your_deepseek_api_key_here" {
        if provider, ok := config.LLM.Providers["deepseek"]; ok {
            provider.APIKey = deepseekKey
            config.LLM.Providers["deepseek"] = provider
        }
    }
    
    if zhipuKey := os.Getenv("ZHIPU_API_KEY"); zhipuKey != "" && zhipuKey != "your_zhipu_api_key_here" {
        if provider, ok := config.LLM.Providers["zhipu"]; ok {
            provider.APIKey = zhipuKey
            config.LLM.Providers["zhipu"] = provider
        }
    }

    if openrouterKey := os.Getenv("OPENROUTER_API_KEY"); openrouterKey != "" && openrouterKey != "your_openrouter_api_key_here" {
        if provider, ok := config.LLM.Providers["openrouter"]; ok {
            provider.APIKey = openrouterKey
            config.LLM.Providers["openrouter"] = provider
        }
    }

    // 从环境变量读取管理员配置
    if adminEmail := os.Getenv("ADMIN_EMAIL"); adminEmail != "" {
        config.Admin.Email = adminEmail
    }
    if adminUsername := os.Getenv("ADMIN_USERNAME"); adminUsername != "" {
        config.Admin.Username = adminUsername
    }
    if adminPassword := os.Getenv("ADMIN_PASSWORD"); adminPassword != "" {
        config.Admin.Password = adminPassword
    }
    // 设置默认值
    if config.Admin.Email == "" {
        config.Admin.Email = "admin@example.com"
    }
    if config.Admin.Username == "" {
        config.Admin.Username = "admin"
    }
    if config.Admin.Password == "" {
        // P0: 不再使用硬编码默认密码，强制要求通过环境变量设置
        fmt.Fprintln(os.Stderr, "[CRITICAL] ADMIN_PASSWORD environment variable is not set! Using a random password. Set ADMIN_PASSWORD for production use.")
        config.Admin.Password = fmt.Sprintf("auto-%d-%d", os.Getpid(), time.Now().UnixNano())
    }

    // 从环境变量读取JWT密钥（优先级高于配置文件）
    if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" && jwtSecret != "change-this-to-a-strong-random-secret-in-production" {
        config.Security.JWTSecret = jwtSecret
    }

    return &config, nil
}

// envVarPattern matches ${VAR_NAME} and ${VAR_NAME:default_value}
var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::([^}]*))?\}`)

// expandEnvVars walks all viper settings and expands ${VAR:default} patterns
func expandEnvVars() {
    for _, key := range viper.AllKeys() {
        val := viper.GetString(key)
        if strings.Contains(val, "${") {
            expanded := expandString(val)
            viper.Set(key, expanded)
        }
    }
}

// expandString replaces all ${VAR:default} occurrences in a string
func expandString(s string) string {
    return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
        parts := envVarPattern.FindStringSubmatch(match)
        if len(parts) < 2 {
            return match
        }
        envKey := parts[1]
        defaultVal := ""
        if len(parts) >= 3 {
            defaultVal = parts[2]
        }
        if envValue := os.Getenv(envKey); envValue != "" {
            return envValue
        }
        return defaultVal
    })
}
