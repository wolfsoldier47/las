package aap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFindJobTemplateID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/controller/v2/job_templates/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "scan-template" {
			t.Errorf("unexpected name query: %v", r.URL.Query())
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			t.Errorf("unexpected auth: %s/%s", user, pass)
		}

		resp := JobTemplatesResponse{
			Results: []JobTemplate{
				{ID: 42, Name: "scan-template", Type: "job_template"},
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/api/controller/v2/", "admin", "secret")
	id, err := client.FindJobTemplateID(context.Background(), "scan-template")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("expected 42, got %d", id)
	}
}

func TestLaunchJobTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/controller/v2/job_templates/42/launch/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		resp := LaunchResponse{ID: 123, Type: "job", Status: "pending"}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/api/controller/v2/", "admin", "secret")
	jobID, err := client.LaunchJobTemplate(context.Background(), 42, LaunchRequest{Limit: "host1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jobID != 123 {
		t.Errorf("expected 123, got %d", jobID)
	}
}

func TestGetJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/controller/v2/jobs/123/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := JobResponse{ID: 123, Status: "successful", Failed: false}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL+"/api/controller/v2/", "admin", "secret")
	job, err := client.GetJob(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Status != "successful" {
		t.Errorf("expected successful, got %s", job.Status)
	}
}
