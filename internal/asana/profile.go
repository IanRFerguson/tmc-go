/*
Copyright © 2024 IAN R FERGUSON IANFERGUSONRVA@gmail.com
*/
package asana

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type storedProfile struct {
	APIKey             string `json:"api_key"`
	WorkspaceGID       string `json:"workspace_gid"`
	WorkspaceName      string `json:"workspace_name"`
	DefaultProjectGID  string `json:"default_project_gid"`
	DefaultProjectName string `json:"default_project_name"`
	// Custom field name -> value as typed (e.g. "today"), applied when creating tasks in the default project
	FieldDefaults map[string]fieldDefault `json:"field_defaults,omitempty"`
}

// fieldDefault holds a stored custom field value; multi-select fields keep one entry per option
// so names containing commas survive. It's written as a plain string when there's a single value.
type fieldDefault []string

func (d fieldDefault) String() string {
	return strings.Join(d, ", ")
}

func (d fieldDefault) MarshalJSON() ([]byte, error) {
	if len(d) == 1 {
		return json.Marshal(d[0])
	}
	return json.Marshal([]string(d))
}

func (d *fieldDefault) UnmarshalJSON(raw []byte) error {
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		*d = fieldDefault{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return err
	}
	*d = many
	return nil
}

// ABOUT - Location of the Asana profile, i.e. ~/.tmc/asana/profile.json
func profilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tmc", "asana", "profile.json"), nil
}

func loadProfile() (storedProfile, error) {
	var profile storedProfile

	path, err := profilePath()
	if err != nil {
		return profile, err
	}

	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return profile, fmt.Errorf("no Asana profile found at %s, run `tmc asana setup` first", path)
	}
	if err != nil {
		return profile, err
	}

	if err := json.Unmarshal(raw, &profile); err != nil {
		return profile, fmt.Errorf("could not parse %s: %w", path, err)
	}

	// Let an environment variable override the stored key (handy for CI or 1Password `op run`)
	if key, ok := os.LookupEnv("ASANA_API_KEY"); ok && key != "" {
		profile.APIKey = key
	}

	return profile, nil
}

// ABOUT - Writes the profile with owner-only permissions since it holds an API key
func saveProfile(profile storedProfile) (string, error) {
	path, err := profilePath()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	raw, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		return "", err
	}
	// WriteFile doesn't change permissions on an existing file
	return path, os.Chmod(path, 0o600)
}
