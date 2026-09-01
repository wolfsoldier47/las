package aap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrJobNotFound is returned when AAP reports that a job ID does not exist.
var ErrJobNotFound = errors.New("aap job not found")

// Client communicates with Ansible Automation Platform (Tower/Controller).
type Client struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

// NewClient creates a new AAP client.
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HealthCheck verifies that AAP is reachable and responds with a healthy status.
func (c *Client) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"ping/", nil)
	if err != nil {
		return fmt.Errorf("create health request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("aap health request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("aap health returned %d: %s", resp.StatusCode, summarizeBody(body))
	}
	return nil
}

// JobTemplate represents an AAP job template listing item.
type JobTemplate struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// JobTemplatesResponse is the response from the AAP job templates list endpoint.
type JobTemplatesResponse struct {
	Results []JobTemplate `json:"results"`
}

// LaunchResponse is the response from launching an AAP job.
type LaunchResponse struct {
	ID     int    `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// JobResponse is the response from getting an AAP job.
type JobResponse struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Failed bool   `json:"failed"`
}

// LaunchRequest contains parameters for launching a job template.
type LaunchRequest struct {
	Limit     string `json:"limit,omitempty"`
	ExtraVars string `json:"extra_vars,omitempty"`
}

// FindJobTemplateID resolves a template name to its AAP ID.
func (c *Client) FindJobTemplateID(ctx context.Context, name string) (int, error) {
	u, err := url.Parse(c.baseURL + "job_templates/")
	if err != nil {
		return 0, fmt.Errorf("parse url: %w", err)
	}
	q := u.Query()
	q.Set("name", name)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		fmt.Println(err)
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		fmt.Println(err)
		return 0, fmt.Errorf("get job templates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return 0, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, summarizeBody(body))
	}

	var result JobTemplatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Results) == 0 {
		return 0, fmt.Errorf("job template %q not found", name)
	}
	return result.Results[0].ID, nil
}

// LaunchJobTemplate launches a job template and returns the AAP job ID.
func (c *Client) LaunchJobTemplate(ctx context.Context, templateID int, req LaunchRequest) (int, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("marshal launch request: %w", err)
	}

	u := fmt.Sprintf("%sjob_templates/%d/launch/", c.baseURL, templateID)
	fmt.Println(u)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	httpReq.SetBasicAuth(c.username, c.password)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("launch job template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, summarizeBody(body))
	}

	var result LaunchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	return result.ID, nil
}

// GetJob fetches the current state of an AAP job.
func (c *Client) GetJob(ctx context.Context, jobID int) (*JobResponse, error) {
	u := fmt.Sprintf("%sjobs/%d/", c.baseURL, jobID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, summarizeBody(body))
	}

	var result JobResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// summarizeBody returns a short, log-safe representation of an AAP error response.
func summarizeBody(body []byte) string {
	if len(body) == 0 {
		return "empty body"
	}
	s := strings.TrimSpace(string(body))
	if strings.HasPrefix(strings.ToLower(s), "<!doctype") || strings.HasPrefix(strings.ToLower(s), "<html") {
		return "<html error page>"
	}
	const max = 200
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

// FormatLimit converts a list of hostnames/patterns to an AAP limit string.
func FormatLimit(hosts []string) string {
	if len(hosts) == 0 {
		return ""
	}
	return strconv.Quote(hosts[0]) // simplistic: AAP limit is a comma-separated pattern
}
