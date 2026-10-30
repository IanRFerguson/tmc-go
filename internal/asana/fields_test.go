package asana

import (
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
)

var priorityField = customField{
	resource:        resource{GID: "99", Name: "Priority"},
	ResourceSubtype: "enum",
	EnumOptions: []enumOption{
		{resource: resource{GID: "a", Name: "Low"}, Enabled: true},
		{resource: resource{GID: "b", Name: "Retired"}, Enabled: false},
		{resource: resource{GID: "c", Name: "High"}, Enabled: true},
	},
}

var tagsField = customField{
	resource:        resource{GID: "98", Name: "Tags"},
	ResourceSubtype: "multi_enum",
	EnumOptions: []enumOption{
		{resource: resource{GID: "x", Name: "Data"}, Enabled: true},
		{resource: resource{GID: "y", Name: "Infra"}, Enabled: true},
	},
}

func TestParseCustomFieldValue(t *testing.T) {
	tests := []struct {
		name    string
		field   customField
		raw     string
		want    any
		wantErr bool
	}{
		{"text", customField{ResourceSubtype: "text"}, "hello", "hello", false},
		{"number", customField{ResourceSubtype: "number"}, "3.5", 3.5, false},
		{"number invalid", customField{ResourceSubtype: "number"}, "lots", nil, true},
		{"date", customField{ResourceSubtype: "date"}, "2026-11-01", map[string]string{"date": "2026-11-01"}, false},
		{"date invalid", customField{ResourceSubtype: "date"}, "Nov 1", nil, true},
		{"enum by number skips disabled options", priorityField, "2", "c", false},
		{"enum by name ignores case", priorityField, "high", "c", false},
		{"enum disabled option", priorityField, "Retired", nil, true},
		{"enum out of range", priorityField, "3", nil, true},
		{"multi_enum", tagsField, "1, infra", []string{"x", "y"}, false},
		{"multi_enum invalid entry", tagsField, "Data, Nope", nil, true},
		{"unsupported type", customField{ResourceSubtype: "people"}, "me", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCustomFieldValue(tt.field, tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestIsPromptableField(t *testing.T) {
	tests := []struct {
		field customField
		want  bool
	}{
		{customField{ResourceSubtype: "text"}, true},
		{customField{ResourceSubtype: "multi_enum"}, true},
		{customField{ResourceSubtype: "people"}, false},
		{customField{ResourceSubtype: "number", IsValueReadOnly: true}, false},
	}

	for _, tt := range tests {
		if got := isPromptableField(tt.field); got != tt.want {
			t.Errorf("isPromptableField(%+v) = %v, want %v", tt.field, got, tt.want)
		}
	}
}

func TestCustomFieldPromptListsEnabledOptions(t *testing.T) {
	prompt := customFieldPrompt(priorityField)
	if !strings.Contains(prompt, "1) Low") || !strings.Contains(prompt, "2) High") {
		t.Errorf("prompt should number enabled options:\n%s", prompt)
	}
	if strings.Contains(prompt, "Retired") {
		t.Errorf("prompt should not list disabled options:\n%s", prompt)
	}
}

// fieldSettingsServer serves a project's custom field settings
func fieldSettingsServer(t *testing.T, fields ...customField) *apiClient {
	return newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/1/custom_field_settings" {
			t.Errorf("path = %q, want /projects/1/custom_field_settings", r.URL.Path)
		}
		settings := make([]customFieldSetting, len(fields))
		for i, f := range fields {
			settings[i] = customFieldSetting{CustomField: f}
		}
		writeData(t, w, settings)
	})
}

func TestCollectCustomFieldsFromFlags(t *testing.T) {
	client := fieldSettingsServer(t, priorityField, tagsField)

	got, err := collectCustomFields(client, "1", nil, []string{"priority=High", "Tags = Data,Infra"}, false)
	if err != nil {
		t.Fatalf("collectCustomFields error: %v", err)
	}
	want := map[string]any{"99": "c", "98": []string{"x", "y"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestCollectCustomFieldsFlagErrors(t *testing.T) {
	tests := map[string]string{
		"missing equals": "Priority",
		"unknown field":  "Owner=me",
		"invalid value":  "Priority=Urgent",
	}

	for name, arg := range tests {
		t.Run(name, func(t *testing.T) {
			client := fieldSettingsServer(t, priorityField)
			if _, err := collectCustomFields(client, "1", nil, []string{arg}, false); err == nil {
				t.Errorf("expected an error for --field %q", arg)
			}
		})
	}
}

func TestCollectCustomFieldsSkipsAPIWhenNothingToDo(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	})

	got, err := collectCustomFields(client, "1", nil, nil, false)
	if err != nil || got != nil {
		t.Errorf("got %v, %v; want nil, nil", got, err)
	}
}

var createdAtField = customField{
	resource:        resource{GID: "97", Name: "Created at"},
	ResourceSubtype: "date",
}

func TestCollectCustomFieldsAppliesDefaults(t *testing.T) {
	client := fieldSettingsServer(t, priorityField, tagsField, createdAtField)
	defaults := map[string]fieldDefault{"Created at": {"today"}, "Priority": {"Low"}, "Tags": {"Data"}, "Removed": {"x"}}
	today, _ := parseDueDate("today")

	got, err := collectCustomFields(client, "1", defaults, []string{"Priority=High", "Tags="}, false)
	if err != nil {
		t.Fatalf("collectCustomFields error: %v", err)
	}
	want := map[string]any{"97": map[string]string{"date": today}, "99": "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestCollectCustomFieldsFlagsBypassBrokenDefaults(t *testing.T) {
	defaults := map[string]fieldDefault{"Priority": {"Retired"}}

	for _, arg := range []string{"Priority=High", "Priority="} {
		client := fieldSettingsServer(t, priorityField)
		if _, err := collectCustomFields(client, "1", defaults, []string{arg}, false); err != nil {
			t.Errorf("--field %q should override the broken default, got %v", arg, err)
		}
	}

	client := fieldSettingsServer(t, priorityField)
	if _, err := collectCustomFields(client, "1", defaults, nil, false); err == nil {
		t.Error("expected an error for a default naming a disabled option")
	}
}

func TestCollectCustomFieldsDoesNotPromptForClearedFields(t *testing.T) {
	// If "Created at" were prompted for (it's listed first), it would take this date
	stdin, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	stdin.WriteString("2026-01-01\n")
	stdin.Seek(0, 0)
	realStdin := os.Stdin
	os.Stdin = stdin
	t.Cleanup(func() { os.Stdin = realStdin })

	client := fieldSettingsServer(t, createdAtField, priorityField)
	defaults := map[string]fieldDefault{"Created at": {"today"}}

	got, err := collectCustomFields(client, "1", defaults, []string{"Created at="}, true)
	if err != nil {
		t.Fatalf("collectCustomFields error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %#v, want no fields set", got)
	}
}

func TestParseStoredFieldValueMatchesEnumsByName(t *testing.T) {
	numbered := customField{
		resource:        resource{GID: "96", Name: "Size"},
		ResourceSubtype: "multi_enum",
		EnumOptions: []enumOption{
			{resource: resource{GID: "s2", Name: "2"}, Enabled: true},
			{resource: resource{GID: "big", Name: "Big"}, Enabled: true},
			{resource: resource{GID: "sm", Name: "Sales, Marketing"}, Enabled: true},
		},
	}

	stored, err := storableFieldValue(numbered, "1, 3")
	if err != nil {
		t.Fatalf("storableFieldValue error: %v", err)
	}
	if want := (fieldDefault{"2", "Sales, Marketing"}); !reflect.DeepEqual(stored, want) {
		t.Fatalf("stored %#v, want %#v", stored, want)
	}

	got, err := parseStoredFieldValue(numbered, stored)
	if err != nil {
		t.Fatalf("parseStoredFieldValue error: %v", err)
	}
	if want := []string{"s2", "sm"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestCollectFieldDefaultsFromFlags(t *testing.T) {
	client := fieldSettingsServer(t, priorityField, tagsField, createdAtField)
	existing := map[string]fieldDefault{"Priority": {"Low"}, "Tags": {"Data"}, "Removed": {"x"}}

	got, err := collectFieldDefaults(client, "1", existing, []string{"created at=Today", "Tags=2, 1", "priority="}, false)
	if err != nil {
		t.Fatalf("collectFieldDefaults error: %v", err)
	}
	want := map[string]fieldDefault{"Created at": {"today"}, "Tags": {"Infra", "Data"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestCollectFieldDefaultsRejectsInvalidValues(t *testing.T) {
	client := fieldSettingsServer(t, createdAtField)
	if _, err := collectFieldDefaults(client, "1", nil, []string{"Created at=someday"}, false); err == nil {
		t.Error("expected an error for an invalid date default")
	}
}
