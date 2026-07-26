package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Orders/internal/customers"
	"Orders/internal/testutil"
)

func TestHandlePutCustomers(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
	}{
		{
			name:        "InvalidJSON",
			body:        `{bad`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "EmptyBody",
			body:        ``,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "UnknownField",
			body:        `[{"uuid":"a","name":"A","foo":"bar"}]`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "WrongContentType",
			body:        `[{"uuid":"a","name":"A"}]`,
			contentType: "text/plain",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name: "BodyTooLarge",
			body: fmt.Sprintf(
				`[{"uuid":"a","name":"%s"}]`,
				strings.Repeat("x", 11<<20),
			),
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "DuplicateUUID",
			body:        `[{"uuid":"a","name":"A"},{"uuid":"a","name":"B"}]`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "Success",
			body:        `[{"uuid":"a","name":"A"}]`,
			contentType: "application/json",
			wantStatus:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.NewTestDB(t)
			app := &App{
				customers: customers.NewStore(db),
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(
				http.MethodPut,
				"/api/customers",
				strings.NewReader(tt.body),
			)
			if tt.contentType != "" {
				r.Header.Set("Content-Type", tt.contentType)
			}

			app.HandlePutCustomers(w, r)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}
