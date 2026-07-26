package app

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"

	"Orders/internal/customers"
)

type CustomerSyncItem struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

func (a *App) HandlePutCustomers(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	defer r.Body.Close()

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		a.BadRequest(w, "Content-Type must be application/json")
		return
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var items []CustomerSyncItem
	if err := dec.Decode(&items); err != nil {
		a.BadRequest(w, "Invalid JSON")
		return
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		a.BadRequest(w, "Unexpected data after JSON body")
		return
	}

	seen := make(map[string]struct{}, len(items))
	models := make([]customers.CustomerSnapshot, len(items))
	for i, item := range items {
		if item.UUID == "" || item.Name == "" {
			a.BadRequest(w, "uuid and name are required")
			return
		}
		if _, exists := seen[item.UUID]; exists {
			a.BadRequest(w, "duplicate uuid: "+item.UUID)
			return
		}
		seen[item.UUID] = struct{}{}
		models[i] = customers.CustomerSnapshot{UUID: item.UUID, Name: item.Name}
	}

	result, err := a.customers.Synchronize(r.Context(), models)
	if err != nil {
		a.InternalError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
