package gotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// recordedRequest — зафиксированный сервером запрос.
type recordedRequest struct {
	key         string
	contentType string
	query       string
	body        messageRequest
}

// newRecordingServer поднимает сервер, который принимает запросы Gotify и
// возвращает статус, зависящий от заголовка X-Gotify-Key (по умолчанию 200).
func newRecordingServer(t *testing.T, statusByKey map[string]int) (*httptest.Server, *[]recordedRequest, *sync.Mutex) {
	t.Helper()

	var mu sync.Mutex
	var recorded []recordedRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg messageRequest
		_ = json.NewDecoder(r.Body).Decode(&msg)

		mu.Lock()
		recorded = append(recorded, recordedRequest{
			key:         r.Header.Get("X-Gotify-Key"),
			contentType: r.Header.Get("Content-Type"),
			query:       r.URL.RawQuery,
			body:        msg,
		})
		mu.Unlock()

		status := http.StatusOK
		if s, ok := statusByKey[r.Header.Get("X-Gotify-Key")]; ok {
			status = s
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)

	return srv, &recorded, &mu
}

func TestSend_DeliversToAllTokens(t *testing.T) {
	srv, recorded, mu := newRecordingServer(t, nil)

	client := New(Config{
		URL:      srv.URL,
		Priority: 7,
		Tokens:   []string{"token-a", "token-b"},
	}, srv.Client())

	err := client.Send(context.Background(), Message{Title: "T", Message: "M"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(*recorded) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(*recorded))
	}

	keys := map[string]bool{}
	for _, req := range *recorded {
		keys[req.key] = true
		if req.contentType != "application/json" {
			t.Errorf("content-type = %q, want application/json", req.contentType)
		}
		if req.query != "" {
			t.Errorf("expected empty query, got %q", req.query)
		}
		if req.body.Title != "T" || req.body.Message != "M" || req.body.Priority != 7 {
			t.Errorf("unexpected body: %+v", req.body)
		}
	}
	if !keys["token-a"] || !keys["token-b"] {
		t.Fatalf("expected both tokens, got %v", keys)
	}
}

func TestSend_ContinuesAfterFailure(t *testing.T) {
	srv, recorded, mu := newRecordingServer(t, map[string]int{"token-a": http.StatusInternalServerError})

	client := New(Config{
		URL:    srv.URL,
		Tokens: []string{"token-a", "token-b"},
	}, srv.Client())

	err := client.Send(context.Background(), Message{Title: "T", Message: "M"})
	if err == nil {
		t.Fatal("expected error when one token fails")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(*recorded) != 2 {
		t.Fatalf("expected both tokens to be attempted, got %d requests", len(*recorded))
	}
}

func TestSend_ErrorDoesNotLeakToken(t *testing.T) {
	const token = "super-secret-token"
	srv, _, _ := newRecordingServer(t, map[string]int{token: http.StatusInternalServerError})

	client := New(Config{
		URL:    srv.URL,
		Tokens: []string{token},
	}, srv.Client())

	err := client.Send(context.Background(), Message{Title: "T", Message: "M"})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaks token: %v", err)
	}
	if strings.Contains(err.Error(), "token=") {
		t.Fatalf("error leaks token query parameter: %v", err)
	}
}
