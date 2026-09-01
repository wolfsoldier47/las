package models

import (
	"time"
)

type AapJobTemplate struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

type AapJobTemplatesResponse struct {
	Results []AapJobTemplate `json:"results"`
}

type AapWorkflowJobResponse struct {
	ID      int        `json:"id"`
	Type    string     `json:"type"`
	Created *time.Time `json:"created"`
	Status  string     `json:"status"`
}

type AapWorkflowJobRequest struct {
	ID        int        `json:"id"`
	Type      string     `json:"type"`
	URL       string     `json:"url"`
	Status    string     `json:"status"`
	Failed    bool       `json:"failed"`
	Started   *time.Time `json:"started,omitempty"`
	Finished  *time.Time `json:"finished,omitempty"`
	ExtraVars string     `json:"extra_vars"`
	Related   Related    `json:"related"`
}

type AapExtraVars struct {
	Limit     string `json:"limit"`
	ExtraVars struct {
		Env            string `json:"env"`
		BackEndBaseURL string `json:"back_end_base_url"`
	} `json:"extra_vars"`
}

type Related struct {
	CreatedBy            string `json:"created_by"`
	Labels               string `json:"labels"`
	Inventory            string `json:"inventory"`
	Project              string `json:"project"`
	Organization         string `json:"organization"`
	Credentials          string `json:"credentials"`
	UnifiedJobTemplate   string `json:"unified_job_template"`
	Stdout               string `json:"stdout"`
	ExecutionEnvironment string `json:"execution_environment"`
	JobEvents            string `json:"job_events"`
	JobHostSummaries     string `json:"job_host_summaries"`
	ActivityStream       string `json:"activity_stream"`
	Notifications        string `json:"notifications"`
	CreateSchedule       string `json:"create_schedule"`
	JobTemplate          string `json:"job_template"`
	Cancel               string `json:"cancel"`
	Relaunch             string `json:"relaunch"`
}

type Limit struct {
	Limit string `json:"limit"`
}

type AapCredential struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type AapJob struct {
	ID                int    `json:"id"`
	Step              string `json:"step"`
	ElapsedTimeMinute int    `json:"elapsed_time_minute"`
	CatalogID         string `json:"catalog_id"`
	HostID            string `json:"host_id"`
}
