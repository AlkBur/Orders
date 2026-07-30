package ui

import "net/http"

const (
	LookupCustomer = "customer"
	LookupProduct  = "product"
)

type LookupResult struct {
	FieldName string
	ID        int64
}

func ReadLookup(r *http.Request) (LookupResult, bool) {
	idStr := r.URL.Query().Get("select_id")
	if idStr == "" {
		return LookupResult{}, false
	}
	id := parseInt64(idStr)
	if id == 0 {
		return LookupResult{}, false
	}
	return LookupResult{
		FieldName: r.URL.Query().Get("select_field"),
		ID:        id,
	}, true
}

func parseInt64(s string) int64 {
	var v int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		v = v*10 + int64(c-'0')
	}
	return v
}
