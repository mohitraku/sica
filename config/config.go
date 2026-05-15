package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Knowledge KnowledgeConfig `mapstructure:"knowledge"`
	AI        AIConfig        `mapstructure:"ai"`
	Calendar  CalendarConfig  `mapstructure:"calendar"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type KnowledgeConfig struct {
	Path string `mapstructure:"path"`
}

type AIConfig struct {
	DefaultProvider string       `mapstructure:"default_provider"`
	Ollama          OllamaConfig `mapstructure:"ollama"`
	Claude          ClaudeConfig `mapstructure:"claude"`
	OpenAI          OpenAIConfig `mapstructure:"openai"`
}

type OllamaConfig struct {
	Host           string `mapstructure:"host"`
	Model          string `mapstructure:"model"`
	EmbeddingModel string `mapstructure:"embedding_model"`
}

type ClaudeConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type OpenAIConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type CalendarConfig struct {
	Outlook OutlookConfig `mapstructure:"outlook"`
}

type OutlookConfig struct {
	ClientID    string `mapstructure:"client_id"`
	RedirectURL string `mapstructure:"redirect_url"`
}

func SicaDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".sica")
}

func Default() *Config {
	sicaDir := SicaDir()
	return &Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8420,
		},
		Database: DatabaseConfig{
			Path: filepath.Join(sicaDir, "data.db"),
		},
		Knowledge: KnowledgeConfig{
			Path: filepath.Join(sicaDir, "knowledge"),
		},
		AI: AIConfig{
			DefaultProvider: "ollama",
			Ollama: OllamaConfig{
				Host:           "http://localhost:11434",
				Model:          "qwen2.5:14b",
				EmbeddingModel: "nomic-embed-text",
			},
			Claude: ClaudeConfig{
				Model: "claude-sonnet-4-6",
			},
			OpenAI: OpenAIConfig{
				Model: "gpt-4o",
			},
		},
		Calendar: CalendarConfig{
			Outlook: OutlookConfig{
				RedirectURL: "http://localhost:8420/auth/outlook/callback",
			},
		},
	}
}

func Load() (*Config, error) {
	sicaDir := SicaDir()
	if err := os.MkdirAll(sicaDir, 0755); err != nil {
		return nil, fmt.Errorf("create sica dir: %w", err)
	}

	cfg := Default()

	configPath := filepath.Join(sicaDir, "config.yaml")

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	v.SetDefault("server", cfg.Server)
	v.SetDefault("database", cfg.Database)
	v.SetDefault("knowledge", cfg.Knowledge)
	v.SetDefault("ai", cfg.AI)
	v.SetDefault("calendar", cfg.Calendar)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		f, ferr := os.Create(configPath)
		if ferr != nil {
			return nil, fmt.Errorf("create default config: %w", ferr)
		}
		f.Close()
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Database.Path = expandPath(cfg.Database.Path)
	cfg.Knowledge.Path = expandPath(cfg.Knowledge.Path)

	return cfg, nil
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
