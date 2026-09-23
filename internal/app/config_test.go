package app

import "testing"

// TestGotifyConfig_Enabled фиксирует условие включения рассылки: сообщения не
// отправляются, если секция отсутствует или url не задан.
func TestGotifyConfig_Enabled(t *testing.T) {
	cases := []struct {
		name string
		cfg  GotifyConfig
		want bool
	}{
		{"absent section", GotifyConfig{}, false},
		{"url only", GotifyConfig{URL: "https://gotify.example.com"}, false},
		{"tokens only", GotifyConfig{Tokens: []string{"a"}}, false},
		{"url and tokens", GotifyConfig{URL: "https://gotify.example.com", Tokens: []string{"a"}}, true},
	}
	for _, c := range cases {
		if got := c.cfg.Enabled(); got != c.want {
			t.Errorf("%s: Enabled() = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestNormalizeGotify проверяет канонизацию секции: url без пробелов и
// завершающего "/", tokens без пустых значений.
func TestNormalizeGotify(t *testing.T) {
	got := normalizeGotify(GotifyConfig{
		URL:    "  https://gotify.example.com/  ",
		Tokens: []string{"", " token-a ", "   ", "token-b"},
	})

	if got.URL != "https://gotify.example.com" {
		t.Fatalf("URL = %q, want %q", got.URL, "https://gotify.example.com")
	}
	if len(got.Tokens) != 2 || got.Tokens[0] != "token-a" || got.Tokens[1] != "token-b" {
		t.Fatalf("Tokens = %v, want [token-a token-b]", got.Tokens)
	}
}

// TestNormalizeGotify_EmptyTokensDisable проверяет, что после нормализации
// секция с одними пустыми токенами остаётся выключенной.
func TestNormalizeGotify_EmptyTokensDisable(t *testing.T) {
	cfg := normalizeGotify(GotifyConfig{
		URL:    "https://gotify.example.com",
		Tokens: []string{"", "   "},
	})
	if cfg.Enabled() {
		t.Fatal("expected empty tokens to disable notifications")
	}
}
