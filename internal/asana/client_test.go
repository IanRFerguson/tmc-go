package asana

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient points an apiClient at a fake Asana server for the duration of the test
func newTestClient(t *testing.T, handler http.HandlerFunc) *apiClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client := newAPIClient("test-key")
	client.baseURL = server.URL
	return client
}

// writeData responds with Asana's {"data": ...} envelope
func writeData(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"data": data}); err != nil {
		t.Fatalf("encoding response: %v", err)
	}
}

func TestMeSendsAuthAndUnwrapsData(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-key")
		}
		if r.URL.Path != "/users/me" {
			t.Errorf("path = %q, want /users/me", r.URL.Path)
		}
		writeData(t, w, map[string]any{
			"gid":        "1",
			"name":       "Test User",
			"workspaces": []map[string]string{{"gid": "10", "name": "TMC"}},
		})
	})

	user, err := client.me()
	if err != nil {
		t.Fatalf("me() error: %v", err)
	}
	if user.Name != "Test User" || len(user.Workspaces) != 1 || user.Workspaces[0].GID != "10" {
		t.Errorf("me() = %+v, want Test User with workspace 10", user)
	}
}

func TestDoReturnsAsanaErrorMessages(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"message":"Not Authorized"},{"message":"Try again"}]}`))
	})

	_, err := client.me()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"401", "Not Authorized", "Try again"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should contain %q", err, want)
		}
	}
}

func TestDoFallsBackToRawBodyOnUnparseableError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("upstream exploded"))
	})

	_, err := client.me()
	if err == nil || !strings.Contains(err.Error(), "upstream exploded") {
		t.Errorf("error = %v, want it to include the raw body", err)
	}
}

func TestFindProjectsKeepsOnlyExactMatches(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces/10/typeahead" {
			t.Errorf("path = %q, want /workspaces/10/typeahead", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("resource_type") != "project" || query.Get("query") != "Ops" {
			t.Errorf("unexpected query %v", query)
		}
		writeData(t, w, []map[string]string{
			{"gid": "1", "name": "Ops"},
			{"gid": "2", "name": "Ops Backlog"},
			{"gid": "3", "name": "ops"},
		})
	})

	projects, err := client.findProjects("10", "Ops")
	if err != nil {
		t.Fatalf("findProjects error: %v", err)
	}
	if len(projects) != 2 || projects[0].GID != "1" || projects[1].GID != "3" {
		t.Errorf("findProjects = %+v, want GIDs 1 and 3", projects)
	}
}

func TestCreateTaskSendsDataEnvelope(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/tasks" {
			t.Errorf("request = %s %s, want POST /tasks", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		var body struct {
			Data map[string]any `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if body.Data["name"] != "Write tests" || body.Data["notes"] != "Lots of them" {
			t.Errorf("unexpected name/notes in %v", body.Data)
		}
		if _, ok := body.Data["assignee"]; ok {
			t.Errorf("assignee should be omitted when empty, got %v", body.Data["assignee"])
		}
		fields, _ := body.Data["custom_fields"].(map[string]any)
		if fields["99"] != "high" {
			t.Errorf("custom_fields = %v, want 99 => high", body.Data["custom_fields"])
		}

		writeData(t, w, map[string]string{
			"gid":           "500",
			"name":          "Write tests",
			"permalink_url": "https://app.asana.com/0/1/500",
		})
	})

	created, err := client.createTask(newTask{
		Name:         "Write tests",
		Notes:        "Lots of them",
		Projects:     []string{"1"},
		CustomFields: map[string]any{"99": "high"},
	})
	if err != nil {
		t.Fatalf("createTask error: %v", err)
	}
	if created.PermalinkURL != "https://app.asana.com/0/1/500" {
		t.Errorf("PermalinkURL = %q", created.PermalinkURL)
	}
}
