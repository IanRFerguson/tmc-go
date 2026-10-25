package asana

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useTempHome isolates profile reads/writes from the real ~/.tmc
func useTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ASANA_API_KEY", "")
	return home
}

func TestSaveAndLoadProfile(t *testing.T) {
	home := useTempHome(t)

	want := storedProfile{
		APIKey:             "secret",
		WorkspaceGID:       "10",
		WorkspaceName:      "TMC",
		DefaultProjectGID:  "1",
		DefaultProjectName: "Ops",
	}
	path, err := saveProfile(want)
	if err != nil {
		t.Fatalf("saveProfile error: %v", err)
	}
	if wantPath := filepath.Join(home, ".tmc", "asana", "profile.json"); path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}

	got, err := loadProfile()
	if err != nil {
		t.Fatalf("loadProfile error: %v", err)
	}
	if got != want {
		t.Errorf("loadProfile = %+v, want %+v", got, want)
	}
}

func TestSaveProfileRestrictsPermissions(t *testing.T) {
	useTempHome(t)

	path, err := saveProfile(storedProfile{APIKey: "secret"})
	if err != nil {
		t.Fatalf("saveProfile error: %v", err)
	}

	// Loosen the file, then save again to confirm permissions are re-tightened
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := saveProfile(storedProfile{APIKey: "secret"}); err != nil {
		t.Fatalf("saveProfile error: %v", err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fileInfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("profile permissions = %o, want 600", perm)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Errorf("profile directory permissions = %o, want 700", perm)
	}
}

func TestLoadProfileMissing(t *testing.T) {
	useTempHome(t)

	_, err := loadProfile()
	if err == nil || !strings.Contains(err.Error(), "tmc asana setup") {
		t.Errorf("error = %v, want a hint to run setup", err)
	}
}

func TestLoadProfileInvalidJSON(t *testing.T) {
	useTempHome(t)

	path, err := profilePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := loadProfile(); err == nil || !strings.Contains(err.Error(), "could not parse") {
		t.Errorf("error = %v, want a parse error", err)
	}
}

func TestLoadProfileEnvOverridesAPIKey(t *testing.T) {
	useTempHome(t)

	if _, err := saveProfile(storedProfile{APIKey: "from-file"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASANA_API_KEY", "from-env")

	got, err := loadProfile()
	if err != nil {
		t.Fatal(err)
	}
	if got.APIKey != "from-env" {
		t.Errorf("APIKey = %q, want from-env", got.APIKey)
	}
}
