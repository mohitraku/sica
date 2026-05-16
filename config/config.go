package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Database  databaseConfig  `mapstructure:"database"`
	Knowledge knowledgeConfig `mapstructure:"knowledge"`
	AI        aiConfig        `mapstructure:"ai"`
}

type databaseConfig struct {
	Path string `mapstructure:"path"`
}

type knowledgeConfig struct {
	Path string `mapstructure:"path"`
}

type aiConfig struct {
	Ollama ollamaConfig `mapstructure:"ollama"`
}

type ollamaConfig struct {
	Host  string `mapstructure:"host"`
	Model string `mapstructure:"model"`
}

func SicaDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".sica")
}

func defaultConfig() *Config {
	sicaDir := SicaDir()
	return &Config{
		Database: databaseConfig{
			Path: filepath.Join(sicaDir, "data.db"),
		},
		Knowledge: knowledgeConfig{
			Path: filepath.Join(sicaDir, "knowledge"),
		},
		AI: aiConfig{
			Ollama: ollamaConfig{
				Host:  "http://localhost:11434",
				Model: "qwen2.5:14b",
			},
		},
	}
}

func Load() (*Config, error) {
	sicaDir := SicaDir()
	if err := os.MkdirAll(sicaDir, 0755); err != nil {
		return nil, fmt.Errorf("create sica dir: %w", err)
	}

	cfg := defaultConfig()
	configPath := filepath.Join(sicaDir, "config.yaml")

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	v.SetDefault("database", cfg.Database)
	v.SetDefault("knowledge", cfg.Knowledge)
	v.SetDefault("ai", cfg.AI)

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
