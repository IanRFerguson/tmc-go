/*
Copyright © 2024 IAN R FERGUSON IANFERGUSONRVA@gmail.com
*/
package asana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://app.asana.com/api/1.0"

type apiClient struct {
	apiKey string
	http   *http.Client
}

type resource struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

type currentUser struct {
	resource
	Workspaces []resource `json:"workspaces"`
}

type enumOption struct {
	resource
	Enabled bool `json:"enabled"`
}

type customField struct {
	resource
	ResourceSubtype string       `json:"resource_subtype"`
	IsValueReadOnly bool         `json:"is_value_read_only"`
	EnumOptions     []enumOption `json:"enum_options"`
}

type customFieldSetting struct {
	CustomField customField `json:"custom_field"`
	IsImportant bool        `json:"is_important"`
}

type newTask struct {
	Name         string         `json:"name"`
	Notes        string         `json:"notes,omitempty"`
	Projects     []string       `json:"projects"`
	Assignee     string         `json:"assignee,omitempty"`
	DueOn        string         `json:"due_on,omitempty"`
	CustomFields map[string]any `json:"custom_fields,omitempty"`
}

type createdTask struct {
	resource
	PermalinkURL string `json:"permalink_url"`
}

func newAPIClient(apiKey string) *apiClient {
	return &apiClient{apiKey: apiKey, http: &http.Client{Timeout: 30 * time.Second}}
}

// ABOUT - Sends a request to the Asana API and unwraps the `data` envelope into `out`
func (c *apiClient) do(method string, path string, query url.Values, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		payload, err := json.Marshal(map[string]any{"data": body})
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(payload)
	}

	endpoint := baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, endpoint, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		var apiErr struct {
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		if json.Unmarshal(raw, &apiErr) == nil && len(apiErr.Errors) > 0 {
			messages := make([]string, len(apiErr.Errors))
			for i, e := range apiErr.Errors {
				messages[i] = e.Message
			}
			return fmt.Errorf("asana API error (%d): %s", resp.StatusCode, strings.Join(messages, "; "))
		}
		return fmt.Errorf("asana API error (%d): %s", resp.StatusCode, string(raw))
	}

	if out == nil {
		return nil
	}
	envelope := struct {
		Data any `json:"data"`
	}{Data: out}
	return json.Unmarshal(raw, &envelope)
}

// ABOUT - Returns the authenticated user, which doubles as an API key check
func (c *apiClient) me() (currentUser, error) {
	var user currentUser
	query := url.Values{"opt_fields": {"name,workspaces.name"}}
	err := c.do(http.MethodGet, "/users/me", query, nil, &user)
	return user, err
}

// ABOUT - Finds projects in a workspace whose name matches exactly (case-insensitive)
func (c *apiClient) findProjects(workspaceGID string, name string) ([]resource, error) {
	var results []resource
	query := url.Values{
		"resource_type": {"project"},
		"query":         {name},
		"count":         {"100"},
		"opt_fields":    {"name"},
	}
	if err := c.do(http.MethodGet, "/workspaces/"+workspaceGID+"/typeahead", query, nil, &results); err != nil {
		return nil, err
	}

	var matches []resource
	for _, project := range results {
		if strings.EqualFold(project.Name, name) {
			matches = append(matches, project)
		}
	}
	return matches, nil
}

// ABOUT - Looks up a single project by GID
func (c *apiClient) getProject(projectGID string) (resource, error) {
	var project resource
	query := url.Values{"opt_fields": {"name"}}
	err := c.do(http.MethodGet, "/projects/"+projectGID, query, nil, &project)
	return project, err
}

// ABOUT - Lists the custom fields attached to a project
func (c *apiClient) customFieldSettings(projectGID string) ([]customFieldSetting, error) {
	var settings []customFieldSetting
	query := url.Values{
		"limit": {"100"},
		"opt_fields": {strings.Join([]string{
			"is_important",
			"custom_field.name",
			"custom_field.resource_subtype",
			"custom_field.is_value_read_only",
			"custom_field.enum_options.name",
			"custom_field.enum_options.enabled",
		}, ",")},
	}
	err := c.do(http.MethodGet, "/projects/"+projectGID+"/custom_field_settings", query, nil, &settings)
	return settings, err
}

func (c *apiClient) createTask(task newTask) (createdTask, error) {
	var created createdTask
	query := url.Values{"opt_fields": {"name,permalink_url"}}
	err := c.do(http.MethodPost, "/tasks", query, task, &created)
	return created, err
}
