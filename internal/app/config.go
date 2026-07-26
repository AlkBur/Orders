package app

import (
	"encoding/json"
	"errors"
	"os"
)

type AuthConfig struct {
	InitialPassword string `json:"initial_password"`
}

type APIConfig struct {
	Key string `json:"key"`
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

	if config.API.Key == "" {
		return nil, errors.New("api.key is required")
	}

	return &config, nil
}
