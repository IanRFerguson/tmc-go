package asana

import (
	"net/http"
	"strings"
	"testing"
)

// projectSearchServer answers typeahead searches from a map of workspace GID -> project names
func projectSearchServer(t *testing.T, projectsByWorkspace map[string][]resource) *apiClient {
	return newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/workspaces/") && strings.HasSuffix(r.URL.Path, "/typeahead"):
			workspaceGID := strings.Split(r.URL.Path, "/")[2]
			writeData(t, w, projectsByWorkspace[workspaceGID])
		case strings.HasPrefix(r.URL.Path, "/projects/"):
			writeData(t, w, resource{GID: strings.TrimPrefix(r.URL.Path, "/projects/"), Name: "By GID"})
		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
			http.NotFound(w, r)
		}
	})
}

func TestFindProjectAcrossWorkspaces(t *testing.T) {
	workspaces := []resource{{GID: "10", Name: "TMC"}, {GID: "20", Name: "Personal"}}
	client := projectSearchServer(t, map[string][]resource{
		"10": {{GID: "1", Name: "Ops"}, {GID: "2", Name: "Shared"}},
		"20": {{GID: "3", Name: "Shared"}},
	})

	t.Run("single match", func(t *testing.T) {
		workspace, project, err := findProjectAcrossWorkspaces(client, workspaces, "", "ops")
		if err != nil {
			t.Fatal(err)
		}
		if workspace.GID != "10" || project.GID != "1" {
			t.Errorf("got %+v / %+v, want workspace 10 project 1", workspace, project)
		}
	})

	t.Run("no match", func(t *testing.T) {
		_, _, err := findProjectAcrossWorkspaces(client, workspaces, "", "Missing")
		if err == nil || !strings.Contains(err.Error(), "no project named") {
			t.Errorf("error = %v, want not found", err)
		}
	})

	t.Run("ambiguous across workspaces", func(t *testing.T) {
		_, _, err := findProjectAcrossWorkspaces(client, workspaces, "", "Shared")
		if err == nil || !strings.Contains(err.Error(), "--workspace") {
			t.Errorf("error = %v, want a hint to use --workspace", err)
		}
	})

	t.Run("workspace filter by name", func(t *testing.T) {
		workspace, project, err := findProjectAcrossWorkspaces(client, workspaces, "personal", "Shared")
		if err != nil {
			t.Fatal(err)
		}
		if workspace.GID != "20" || project.GID != "3" {
			t.Errorf("got %+v / %+v, want workspace 20 project 3", workspace, project)
		}
	})

	t.Run("workspace filter by GID", func(t *testing.T) {
		_, project, err := findProjectAcrossWorkspaces(client, workspaces, "10", "Shared")
		if err != nil {
			t.Fatal(err)
		}
		if project.GID != "2" {
			t.Errorf("project = %+v, want GID 2", project)
		}
	})
}

func TestFindProjectInWorkspace(t *testing.T) {
	client := projectSearchServer(t, map[string][]resource{
		"10": {{GID: "1", Name: "Ops"}, {GID: "2", Name: "Dupe"}, {GID: "3", Name: "Dupe"}},
	})

	t.Run("by name", func(t *testing.T) {
		project, err := findProjectInWorkspace(client, "10", "Ops")
		if err != nil || project.GID != "1" {
			t.Errorf("got %+v, %v; want GID 1", project, err)
		}
	})

	t.Run("numeric input is treated as a GID", func(t *testing.T) {
		project, err := findProjectInWorkspace(client, "10", "12345")
		if err != nil || project.GID != "12345" {
			t.Errorf("got %+v, %v; want GID 12345", project, err)
		}
	})

	t.Run("no match", func(t *testing.T) {
		if _, err := findProjectInWorkspace(client, "10", "Missing"); err == nil {
			t.Error("expected an error")
		}
	})

	t.Run("duplicate names list GIDs", func(t *testing.T) {
		_, err := findProjectInWorkspace(client, "10", "Dupe")
		if err == nil || !strings.Contains(err.Error(), "2, 3") {
			t.Errorf("error = %v, want it to list GIDs 2, 3", err)
		}
	})
}
