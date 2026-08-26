package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type AuthConfig struct {
	InitialPassword string `json:"initial_password"`
}

// LimitConfig описывает лимит: requests запросов за window_sec секунд.
type LimitConfig struct {
	Requests  int `json:"requests"`
	WindowSec int `json:"window_sec"`
}

// RateLimitConfig — настройки rate limiter'ов.
type RateLimitConfig struct {
	LoginByIP      LimitConfig `json:"login_by_ip"`
	LoginByAccount LimitConfig `json:"login_by_account"`
	IntegrationAPI LimitConfig `json:"integration_api"`
}

type Config struct {
	HTTPAddress       string          `json:"http_address"`
	DatabasePath      string          `json:"database_path"`
	FilesDatabasePath string          `json:"files_database_path"`
	BasePath          string          `json:"base_path"`
	Secret            string          `json:"secret"`
	Auth              AuthConfig      `json:"auth"`
	RateLimit         RateLimitConfig `json:"rate_limit"`
}

// defaultRateLimit применяется, когда соответствующая секция не задана или
// requests <= 0 — старые конфиги продолжают работать без изменений.
var defaultRateLimit = RateLimitConfig{
	LoginByIP:      LimitConfig{Requests: 10, WindowSec: 60},
	LoginByAccount: LimitConfig{Requests: 5, WindowSec: 600},
	IntegrationAPI: LimitConfig{Requests: 120, WindowSec: 60},
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

	// Старые конфиги без files_database_path продолжают работать:
	// файловая база по умолчанию лежит рядом с основной.
	if config.DatabasePath != "" && config.FilesDatabasePath == "" {
		config.FilesDatabasePath = filepath.Join(filepath.Dir(config.DatabasePath), "files.db")
	}

	config.BasePath = NormalizeBasePath(config.BasePath)

	config.applyRateLimitDefaults()

	return &config, nil
}

// NormalizeBasePath приводит base_path из конфига к каноническому виду.
//
// Базовый префикс — это всегда внутренний абсолютный путь приложения,
// начинающийся с "/" и не заканчивающийся "/".
//
//	""      → ""
//	"/"     → ""
//	"app"   → "/app"
//	"/app/" → "/app"
//	" /app/ " → "/app"
//
// Пустой результат означает «приложение опубликовано в корне» —
// префикс не добавляется ни к одному URL.
func NormalizeBasePath(basePath string) string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" || basePath == "/" {
		return ""
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	return strings.TrimRight(basePath, "/")
}

// applyRateLimitDefaults заменяет лимиты, отсутствующие в конфиге, значениями
// по умолчанию. Лимит считается отсутствующим, если requests или window_sec
// не заданы (<= 0) — старые конфиги без секции rate_limit продолжают работать.
func (c *Config) applyRateLimitDefaults() {
	if c.RateLimit.LoginByIP.Requests <= 0 || c.RateLimit.LoginByIP.WindowSec <= 0 {
		c.RateLimit.LoginByIP = defaultRateLimit.LoginByIP
	}
	if c.RateLimit.LoginByAccount.Requests <= 0 || c.RateLimit.LoginByAccount.WindowSec <= 0 {
		c.RateLimit.LoginByAccount = defaultRateLimit.LoginByAccount
	}
	if c.RateLimit.IntegrationAPI.Requests <= 0 || c.RateLimit.IntegrationAPI.WindowSec <= 0 {
		c.RateLimit.IntegrationAPI = defaultRateLimit.IntegrationAPI
	}
}
