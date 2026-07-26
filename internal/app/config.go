package app

import (
	"encoding/json"
	"os"
)

type Config struct {
	HTTPAddress  string `json:"http_address"`
	DatabasePath string `json:"database_path"`
	Secret       string `json:"secret"`
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

	return &config, nil
}
