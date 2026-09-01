package servicenow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateIncident(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/now/table/incident" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			t.Errorf("unexpected auth: %s/%s", user, pass)
		}

		resp := CreateIncidentResponse{}
		resp.Result.Number = "INC0010001"
		resp.Result.SysID = "abc123"
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "admin", "secret")
	req := CreateIncidentRequest{
		ShortDescription: "test incident",
		Description:      "deviation detected",
		Urgency:          2,
		Impact:           2,
	}
	number, url, err := client.CreateIncident(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if number != "INC0010001" {
		t.Errorf("expected INC0010001, got %s", number)
	}
	if url == "" {
		t.Error("expected non-empty url")
	}
}
