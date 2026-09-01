package servicenow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client communicates with ServiceNow.
type Client struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

// NewClient creates a new ServiceNow client.
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

// CreateIncidentRequest is the payload for creating a ServiceNow incident.
type CreateIncidentRequest struct {
	ShortDescription string `json:"short_description"`
	Description      string `json:"description"`
	Urgency          int    `json:"urgency"`
	Impact           int    `json:"impact"`
}

// CreateIncidentResponse is the response from creating a ServiceNow incident.
type CreateIncidentResponse struct {
	Result struct {
		Number  string `json:"number"`
		SysID   string `json:"sys_id"`
		OpenURL string `json:"opened_by_link,omitempty"`
	} `json:"result"`
}

// CreateIncident opens a new ServiceNow incident and returns its number and URL.
func (c *Client) CreateIncident(ctx context.Context, req CreateIncidentRequest) (string, string, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("marshal request: %w", err)
	}

	u := c.baseURL + "/api/now/table/incident"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	httpReq.SetBasicAuth(c.username, c.password)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("create incident: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result CreateIncidentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}

	ticketURL := fmt.Sprintf("%s/incident.do?sys_id=%s", c.baseURL, result.Result.SysID)
	return result.Result.Number, ticketURL, nil
}
