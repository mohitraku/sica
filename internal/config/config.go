package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DataDir string `json:"data_dir,omitempty"`
}

func dir() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "sica"), nil
}

func Load() (Config, error) {
	d, err := dir()
	if err != nil {
		return Config{}, err
	}
	f, err := os.Open(filepath.Join(d, "config.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}
	defer f.Close()
	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Save(cfg Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(d, "config.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}
