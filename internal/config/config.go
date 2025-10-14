package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	OpenAI   OpenAIConfig
	App      AppConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	URL            string
	AnonKey        string
	ServiceRoleKey string
}

type OpenAIConfig struct {
	APIKey          string
	EmbeddingModel  string
	CompletionModel string
}

type AppConfig struct {
	ChunkSize     int
	ChunkOverlap  int
	TopK          int
	ContextLimit  int
	CacheDuration time.Duration
	Debug         bool
	LogLevel      string
}

func Load() (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			// Only log if it's not just a missing file
			log.Printf("Warning: Could not load .env file: %v", err)
		}
	}

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "localhost"),
		},
		Database: DatabaseConfig{
			URL:            getEnv("SUPABASE_URL", ""),
			AnonKey:        getEnv("SUPABASE_ANON_KEY", ""),
			ServiceRoleKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		},
		OpenAI: OpenAIConfig{
			APIKey:          getEnv("OPENAI_API_KEY", ""),
			EmbeddingModel:  getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
			CompletionModel: getEnv("OPENAI_COMPLETION_MODEL", "gpt-4o-mini"),
		},
		App: AppConfig{
			ChunkSize:     getEnvAsInt("CHUNK_SIZE", 900),
			ChunkOverlap:  getEnvAsInt("CHUNK_OVERLAP", 150),
			TopK:          getEnvAsInt("TOP_K", 5),
			ContextLimit:  getEnvAsInt("CONTEXT_LIMIT", 4000),
			CacheDuration: getEnvAsDuration("CACHE_DURATION", "24h"),
			Debug:         getEnvAsBool("DEBUG", false),
			LogLevel:      getEnv("LOG_LEVEL", "info"),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}
