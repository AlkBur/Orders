package app

import (
	"encoding/json"
	"errors"
	"os"
)

type AuthConfig struct {
	InitialPassword string `json:"initial_password"`
}

type APIKeyConfig struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type APIConfig struct {
	Keys []APIKeyConfig `json:"keys"`
}

type Config struct {
	HTTPAddress  string     `json:"http_address"`
	DatabasePath string     `json:"database_path"`
	Secret       string     `json:"secret"`
	Auth         AuthConfig `json:"auth"`
	API          APIConfig  `json:"api"`
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	if config.Auth.InitialPassword == "" {
		return nil, errors.New("auth.initial_password is required")
	}

	if len(config.API.Keys) == 0 {
		return nil, errors.New("at least one api.keys entry is required")
	}

	for i, k := range config.API.Keys {
		if k.Name == "" {
			return nil, errors.New("api.keys.name is required")
		}
		if k.Key == "" {
			return nil, errors.New("api.keys.key is required")
		}
		_ = i
	}

	return &config, nil
}
