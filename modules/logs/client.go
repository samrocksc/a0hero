// Package logs provides the Auth0 Logs module.
package logs

import (
	"context"
	"fmt"
	"time"

	"github.com/samrocksc/a0hero/client"
)

// LogEvent represents a single Auth0 log event.
type LogEvent struct {
	ID       string         `json:"log_id,omitempty"`
	Date     string         `json:"date"`
	Type     string         `json:"type"`
	IP       string         `json:"ip,omitempty"`
	UserID   string         `json:"user_id,omitempty"`
	UserName string         `json:"user_name,omitempty"`
	ClientID string         `json:"client_id,omitempty"`
	Details  map[string]any `json:"details,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}

// LogEventTypeCode maps Auth0 log type codes to human-readable descriptions.
var LogEventTypeCode = map[string]string{
	"felo": "Failed Login",
	"fai":  "Failed Login (Invalid Credentials)",
	"slo":  "Successful Logout",
	"suca": "Successful Login",
	"pacu": "Password Change",
	"fc":   "Failed by Connector",
	"w":    "Warning",
	"sep":  "Successful Signup",
	"fep":  "Failed Signup",
	"sai":  "Successful API Call",
	"faii": "Failed API Call",
	"ss":   "Successful Signup",
	"fs":   "Failed Signup",
	"cs":   "Code Sent",
	"cls":  "Code/Link Sent",
	"sv":   "Verification",
}

// Describe returns a human-readable description of the log type.
func (e LogEvent) Describe() string {
	if desc, ok := LogEventTypeCode[e.Type]; ok {
		return desc
	}
	return e.Type
}

// FormatDate parses the log date and returns a formatted string.
func (e LogEvent) FormatDate() string {
	t, err := time.Parse(time.RFC3339, e.Date)
	if err != nil {
		return e.Date
	}
	return t.Format("2006-01-02 15:04:05")
}

// Row converts a LogEvent into a concise one-liner for the table.
func (e LogEvent) Row() []string {
	return []string{e.FormatDate(), e.Type, e.Describe(), e.UserName, e.IP}
}

// Columns returns the column headers for a log events table.
func Columns() []string {
	return []string{"Time", "Type", "Event", "User", "IP"}
}

// Client wraps the Auth0 /api/v2/logs endpoints.
type LogsClient struct {
	c *client.Client
}

// New creates a new Logs module client.
func New(c *client.Client) *LogsClient {
	return &LogsClient{c: c}
}

// List returns log events from the Auth0 tenant.
// Parameters from and take follow Auth0's cursor-based pagination.
func (lc *LogsClient) List(ctx context.Context, from string, take int) ([]LogEvent, error) {
	query := fmt.Sprintf("take=%d", take)
	if from != "" {
		query += "&from=" + from
	}
	var result []LogEvent
	if err := lc.c.GetWithQuery(ctx, "/api/v2/logs", query, &result); err != nil {
		return nil, fmt.Errorf("logs: List: %w", err)
	}
	return result, nil
}

// Get returns a single log event by ID.
func (lc *LogsClient) Get(ctx context.Context, logID string) (*LogEvent, error) {
	var event LogEvent
	if err := lc.c.Get(ctx, "/api/v2/logs/"+logID, &event); err != nil {
		return nil, fmt.Errorf("logs: Get: %w", err)
	}
	return &event, nil
}
