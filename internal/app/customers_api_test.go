package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Orders/internal/customers"
	"Orders/internal/testutil"
)

type CustomerTableRow struct {
	UUID   string
	Name   string
	Active bool
}

type Step struct {
	Body   string
	Result customers.SyncResult
}

type Scenario struct {
	Name     string
	Steps    []Step
	Expected []CustomerTableRow
}

func scanCustomer(rows *sql.Rows) (CustomerTableRow, error) {
	var c CustomerTableRow
	err := rows.Scan(&c.UUID, &c.Name, &c.Active)
	return c, err
}

func TestCustomersAPI(t *testing.T) {
	db := testutil.NewTestDB(t, NewSchema())
	app := &App{
		customers:    customers.NewStore(db),
		integrations: map[string]*Integration{"test-key": {Name: "Test"}},
	}

	put := func(body string) (customers.SyncResult, *httptest.ResponseRecorder) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPut, "/api/integration/customers",
			strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-API-Key", "test-key")
		app.RequireIntegration(http.HandlerFunc(app.HandlePutCustomers)).ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result customers.SyncResult
		json.NewDecoder(w.Body).Decode(&result)
		return result, w
	}

	query := "SELECT uuid, name, active FROM customers ORDER BY uuid"

	scenarios := []Scenario{
		{
			Name: "FirstImport",
			Steps: []Step{
				{
					Body:   `[{"uuid":"a","name":"A"},{"uuid":"b","name":"B"}]`,
					Result: customers.SyncResult{Inserted: 2},
				},
			},
			Expected: []CustomerTableRow{
				{UUID: "a", Name: "A", Active: true},
				{UUID: "b", Name: "B", Active: true},
			},
		},
		{
			Name: "Idempotent",
			Steps: []Step{
				{
					Body:   `[{"uuid":"a","name":"A"}]`,
					Result: customers.SyncResult{Inserted: 1},
				},
				{
					Body:   `[{"uuid":"a","name":"A"}]`,
					Result: customers.SyncResult{Inserted: 0, Updated: 0, Deactivated: 0},
				},
			},
			Expected: []CustomerTableRow{
				{UUID: "a", Name: "A", Active: true},
			},
		},
		{
			Name: "Update",
			Steps: []Step{
				{
					Body:   `[{"uuid":"a","name":"A"}]`,
					Result: customers.SyncResult{Inserted: 1},
				},
				{
					Body:   `[{"uuid":"a","name":"A-new"}]`,
					Result: customers.SyncResult{Updated: 1},
				},
			},
			Expected: []CustomerTableRow{
				{UUID: "a", Name: "A-new", Active: true},
			},
		},
		{
			Name: "ActivationLifecycle",
			Steps: []Step{
				{
					Body:   `[{"uuid":"a","name":"A"},{"uuid":"b","name":"B"}]`,
					Result: customers.SyncResult{Inserted: 2},
				},
				{
					Body:   `[{"uuid":"b","name":"B"}]`,
					Result: customers.SyncResult{Deactivated: 1},
				},
				{
					Body:   `[{"uuid":"a","name":"A"},{"uuid":"b","name":"B"}]`,
					Result: customers.SyncResult{Updated: 1},
				},
			},
			Expected: []CustomerTableRow{
				{UUID: "a", Name: "A", Active: true},
				{UUID: "b", Name: "B", Active: true},
			},
		},
		{
			Name: "EmptyList",
			Steps: []Step{
				{
					Body:   `[{"uuid":"a","name":"A"},{"uuid":"b","name":"B"}]`,
					Result: customers.SyncResult{Inserted: 2},
				},
				{
					Body:   `[]`,
					Result: customers.SyncResult{Deactivated: 2},
				},
			},
			Expected: []CustomerTableRow{
				{UUID: "a", Name: "A", Active: false},
				{UUID: "b", Name: "B", Active: false},
			},
		},
		{
			Name: "Complex",
			Steps: []Step{
				{
					Body:   `[{"uuid":"a","name":"A"},{"uuid":"b","name":"B"},{"uuid":"c","name":"C"}]`,
					Result: customers.SyncResult{Inserted: 3},
				},
				{
					Body:   `[{"uuid":"a","name":"A-new"},{"uuid":"c","name":"C"},{"uuid":"d","name":"D"}]`,
					Result: customers.SyncResult{Inserted: 1, Updated: 1, Deactivated: 1},
				},
			},
			Expected: []CustomerTableRow{
				{UUID: "a", Name: "A-new", Active: true},
				{UUID: "b", Name: "B", Active: false},
				{UUID: "c", Name: "C", Active: true},
				{UUID: "d", Name: "D", Active: true},
			},
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			db.Exec("DELETE FROM customers")

			for i, step := range sc.Steps {
				result, _ := put(step.Body)

				if result != step.Result {
					t.Fatalf("step %d result mismatch:\nexpected: %+v\ngot:      %+v",
						i, step.Result, result)
				}
			}

			got, err := testutil.LoadRows(
				context.Background(),
				db,
				query,
				scanCustomer,
			)
			if err != nil {
				t.Fatal(err)
			}

			testutil.AssertSlice(t, sc.Expected, got)
		})
	}
}
