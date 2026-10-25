package memberlibrary

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/mehanizm/airtable"
)

// newTestAirtable serves a paginated "domains" table: each element of pages is one page of results
func newTestAirtable(t *testing.T, pages [][]string, onPost func(r *http.Request)) airtable.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/db/tbl" {
			t.Errorf("path = %q, want /db/tbl", r.URL.Path)
		}

		if r.Method == http.MethodPost {
			if onPost != nil {
				onPost(r)
			}
			w.Write([]byte(`{"records":[]}`))
			return
		}

		page := 0
		if offset := r.URL.Query().Get("offset"); offset != "" {
			page = int(offset[0] - '0')
		}
		records := []map[string]any{}
		for _, domain := range pages[page] {
			records = append(records, map[string]any{"fields": map[string]any{"domains": domain}})
		}
		body := map[string]any{"records": records}
		if page+1 < len(pages) {
			body["offset"] = string(rune('0' + page + 1))
		}
		json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(server.Close)

	client := airtable.NewClient("test-key")
	if err := client.SetBaseURL(server.URL); err != nil {
		t.Fatal(err)
	}
	client.SetRateLimit(1000)
	return *client
}

func TestBuildDomainArrayFollowsPagination(t *testing.T) {
	client := newTestAirtable(t, [][]string{{"a.org", "b.org"}, {"c.org"}}, nil)

	got := buildDomainArray(client, "db", "tbl")
	want := []string{"a.org", "b.org", "c.org"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildDomainArray = %v, want %v", got, want)
	}
}

func TestCheckAirtable(t *testing.T) {
	client := newTestAirtable(t, [][]string{{"a.org"}, {"movementcooperative.org"}}, nil)

	if !checkAirtable(client, "movementcooperative.org", "db", "tbl") {
		t.Error("expected movementcooperative.org to exist")
	}
	if checkAirtable(client, "missing.org", "db", "tbl") {
		t.Error("expected missing.org to not exist")
	}
}

func TestAddToAirtableSkipsExistingDomain(t *testing.T) {
	client := newTestAirtable(t, [][]string{{"a.org"}}, func(r *http.Request) {
		t.Error("should not write a domain that already exists")
	})

	addToAirtable(client, "a.org", "db", "tbl")
}

func TestAddToAirtableWritesNewDomain(t *testing.T) {
	var posted []string
	client := newTestAirtable(t, [][]string{{"a.org"}}, func(r *http.Request) {
		var body struct {
			Records []struct {
				Fields map[string]string `json:"fields"`
			} `json:"records"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		for _, record := range body.Records {
			posted = append(posted, record.Fields["domains"])
		}
	})

	addToAirtable(client, "new.org", "db", "tbl")
	if !reflect.DeepEqual(posted, []string{"new.org"}) {
		t.Errorf("posted %v, want [new.org]", posted)
	}
}

func TestCheckEnvironment(t *testing.T) {
	t.Run("explicit flag wins", func(t *testing.T) {
		t.Setenv("AIRTABLE_TABLE", "from-env")
		if got := checkEnvironment("from-flag", "AIRTABLE_TABLE", "DEFAULT"); got != "from-flag" {
			t.Errorf("got %q, want from-flag", got)
		}
	})

	t.Run("default falls back to environment", func(t *testing.T) {
		t.Setenv("AIRTABLE_TABLE", "from-env")
		if got := checkEnvironment("DEFAULT", "AIRTABLE_TABLE", "DEFAULT"); got != "from-env" {
			t.Errorf("got %q, want from-env", got)
		}
	})

	t.Run("missing environment panics", func(t *testing.T) {
		// Setenv registers a restore on cleanup, so it's safe to unset afterwards
		t.Setenv("AIRTABLE_TABLE", "")
		os.Unsetenv("AIRTABLE_TABLE")

		defer func() {
			if recover() == nil {
				t.Error("expected a panic")
			}
		}()
		checkEnvironment("DEFAULT", "AIRTABLE_TABLE", "DEFAULT")
	})
}
