/*
Copyright © 2024 IAN R FERGUSON IANFERGUSONRVA@gmail.com
*/
package asana

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ABOUT - Builds the custom_fields payload from --field flags and, optionally, interactive prompts
func collectCustomFields(client *apiClient, projectGID string, fieldArgs []string, interactive bool) (map[string]any, error) {
	if len(fieldArgs) == 0 && !interactive {
		return nil, nil
	}

	settings, err := client.customFieldSettings(projectGID)
	if err != nil {
		return nil, fmt.Errorf("could not load the project's custom fields: %w", err)
	}

	values := map[string]any{}

	for _, arg := range fieldArgs {
		name, raw, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("--field %q should look like \"Name=value\"", arg)
		}
		field, found := findCustomField(settings, strings.TrimSpace(name))
		if !found {
			return nil, fmt.Errorf("project has no custom field named %q", name)
		}
		value, err := parseCustomFieldValue(field, strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("--field %q: %w", arg, err)
		}
		values[field.GID] = value
	}

	if !interactive {
		return values, nil
	}

	reader := bufio.NewReader(os.Stdin)
	printedHeader := false
	for _, setting := range settings {
		field := setting.CustomField
		if _, alreadySet := values[field.GID]; alreadySet || !isPromptableField(field) {
			continue
		}
		if !printedHeader {
			fmt.Println("Custom fields (leave blank to skip):")
			printedHeader = true
		}

		for {
			raw, err := promptLine(reader, customFieldPrompt(field))
			if err != nil {
				return nil, err
			}
			if raw == "" {
				break
			}
			value, err := parseCustomFieldValue(field, raw)
			if err != nil {
				fmt.Printf("  %s\n", err)
				continue
			}
			values[field.GID] = value
			break
		}
	}

	return values, nil
}

func findCustomField(settings []customFieldSetting, name string) (customField, bool) {
	for _, setting := range settings {
		if strings.EqualFold(setting.CustomField.Name, name) {
			return setting.CustomField, true
		}
	}
	return customField{}, false
}

// ABOUT - Field types we know how to collect from a terminal (people, formula, etc. are skipped)
func isPromptableField(field customField) bool {
	if field.IsValueReadOnly {
		return false
	}
	switch field.ResourceSubtype {
	case "text", "number", "date", "enum", "multi_enum":
		return true
	}
	return false
}

func customFieldPrompt(field customField) string {
	switch field.ResourceSubtype {
	case "enum", "multi_enum":
		var b strings.Builder
		fmt.Fprintf(&b, "  %s:\n", field.Name)
		for i, option := range enabledOptions(field) {
			fmt.Fprintf(&b, "    %d) %s\n", i+1, option.Name)
		}
		if field.ResourceSubtype == "multi_enum" {
			b.WriteString("  Choose one or more (comma-separated): ")
		} else {
			b.WriteString("  Choose one: ")
		}
		return b.String()
	case "date":
		return fmt.Sprintf("  %s (YYYY-MM-DD, today, tomorrow): ", field.Name)
	default:
		return fmt.Sprintf("  %s (%s): ", field.Name, field.ResourceSubtype)
	}
}

// ABOUT - Converts user input into the shape the Asana API expects for each field type
func parseCustomFieldValue(field customField, raw string) (any, error) {
	switch field.ResourceSubtype {
	case "text":
		return raw, nil
	case "number":
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", raw)
		}
		return n, nil
	case "date":
		date, err := parseDueDate(raw)
		if err != nil {
			return nil, err
		}
		return map[string]string{"date": date}, nil
	case "enum":
		option, err := matchEnumOption(field, raw)
		if err != nil {
			return nil, err
		}
		return option.GID, nil
	case "multi_enum":
		var gids []string
		for _, part := range strings.Split(raw, ",") {
			option, err := matchEnumOption(field, strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			gids = append(gids, option.GID)
		}
		return gids, nil
	}
	return nil, fmt.Errorf("custom field %q has unsupported type %q", field.Name, field.ResourceSubtype)
}

// ABOUT - Matches an enum option by its list number or name
func matchEnumOption(field customField, raw string) (enumOption, error) {
	options := enabledOptions(field)
	if i, err := strconv.Atoi(raw); err == nil && i >= 1 && i <= len(options) {
		return options[i-1], nil
	}
	for _, option := range options {
		if strings.EqualFold(option.Name, raw) {
			return option, nil
		}
	}
	return enumOption{}, fmt.Errorf("%q is not an option for %s", raw, field.Name)
}

func enabledOptions(field customField) []enumOption {
	var options []enumOption
	for _, option := range field.EnumOptions {
		if option.Enabled {
			options = append(options, option)
		}
	}
	return options
}
