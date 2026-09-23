// Package gotify отправляет уведомления в Gotify. Пакет не знает о доменной
// модели приложения: он получает готовые заголовок и текст и занимается только
// транспортом Gotify.
package gotify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Config — параметры клиента. Один URL и один priority применяются ко всем
// tokens: одно уведомление уходит в каждое приложение.
type Config struct {
	URL      string
	Priority int
	Tokens   []string
}

// Message — готовое к отправке уведомление.
type Message struct {
	Title   string
	Message string
}

type Client struct {
	baseURL  string
	priority int
	tokens   []string
	http     *http.Client
}

// New создаёт клиент Gotify. Если httpClient не задан, используется
// http.DefaultClient.
func New(cfg Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:  cfg.URL,
		priority: cfg.Priority,
		tokens:   cfg.Tokens,
		http:     httpClient,
	}
}

// messageRequest — тело запроса POST /message.
type messageRequest struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
}

// Send рассылает сообщение во все приложения Gotify. Ошибка одного токена не
// мешает остальным; ошибки объединяются через errors.Join. Токен передаётся
// заголовком X-Gotify-Key и никогда не попадает в URL или текст ошибки.
func (c *Client) Send(ctx context.Context, msg Message) error {
	var errs []error
	for i, token := range c.tokens {
		if err := c.sendOne(ctx, token, msg); err != nil {
			errs = append(errs, fmt.Errorf("token #%d: %w", i+1, err))
		}
	}
	return errors.Join(errs...)
}

func (c *Client) sendOne(ctx context.Context, token string, msg Message) error {
	body, err := json.Marshal(messageRequest{
		Title:    msg.Title,
		Message:  msg.Message,
		Priority: c.priority,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/message", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gotify-Key", token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
