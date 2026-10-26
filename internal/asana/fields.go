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

// ABOUT - Builds the custom_fields payload from profile defaults, --field flags and, optionally, interactive prompts
func collectCustomFields(client *apiClient, projectGID string, defaults map[string]string, fieldArgs []string, interactive bool) (map[string]any, error) {
	if len(defaults) == 0 && len(fieldArgs) == 0 && !interactive {
		return nil, nil
	}

	settings, err := client.customFieldSettings(projectGID)
	if err != nil {
		return nil, fmt.Errorf("could not load the project's custom fields: %w", err)
	}

	values := map[string]any{}

	for name, raw := range defaults {
		field, found := findCustomField(settings, name)
		if !found {
			fmt.Fprintf(os.Stderr, "Skipping default for %q, the project no longer has that field\n", name)
			continue
		}
		value, err := parseCustomFieldValue(field, raw)
		if err != nil {
			return nil, fmt.Errorf("default for %s: %w", field.Name, err)
		}
		values[field.GID] = value
	}

	for _, arg := range fieldArgs {
		field, raw, err := parseFieldArg(settings, arg)
		if err != nil {
			return nil, err
		}
		// An empty value (e.g. --field "Created at=") clears a default
		if raw == "" {
			delete(values, field.GID)
			continue
		}
		value, err := parseCustomFieldValue(field, raw)
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

// ABOUT - Builds the field defaults stored in the profile from --field flags and, optionally, interactive prompts
func collectFieldDefaults(client *apiClient, projectGID string, existing map[string]string, fieldArgs []string, interactive bool) (map[string]string, error) {
	if len(existing) == 0 && len(fieldArgs) == 0 && !interactive {
		return nil, nil
	}

	settings, err := client.customFieldSettings(projectGID)
	if err != nil {
		return nil, fmt.Errorf("could not load the project's custom fields: %w", err)
	}

	// Carry over existing defaults, dropping any for fields the project no longer has
	defaults := map[string]string{}
	for name, raw := range existing {
		if field, found := findCustomField(settings, name); found {
			defaults[field.Name] = raw
		}
	}

	setByFlag := map[string]bool{}
	for _, arg := range fieldArgs {
		field, raw, err := parseFieldArg(settings, arg)
		if err != nil {
			return nil, err
		}
		setByFlag[field.GID] = true
		if raw == "" {
			delete(defaults, field.Name)
			continue
		}
		stored, err := storableFieldValue(field, raw)
		if err != nil {
			return nil, fmt.Errorf("--field %q: %w", arg, err)
		}
		defaults[field.Name] = stored
	}

	if interactive {
		reader := bufio.NewReader(os.Stdin)
		printedHeader := false
		for _, setting := range settings {
			field := setting.CustomField
			if setByFlag[field.GID] || !isPromptableField(field) {
				continue
			}
			if !printedHeader {
				fmt.Println("Default custom field values (leave blank to keep the current one, - to clear):")
				printedHeader = true
			}

			prompt := customFieldPrompt(field)
			if current, ok := defaults[field.Name]; ok {
				prompt = strings.TrimSuffix(prompt, ": ") + fmt.Sprintf(" [%s]: ", current)
			}
			for {
				raw, err := promptLine(reader, prompt)
				if err != nil {
					return nil, err
				}
				if raw == "" {
					break
				}
				if raw == "-" {
					delete(defaults, field.Name)
					break
				}
				stored, err := storableFieldValue(field, raw)
				if err != nil {
					fmt.Printf("  %s\n", err)
					continue
				}
				defaults[field.Name] = stored
				break
			}
		}
	}

	if len(defaults) == 0 {
		return nil, nil
	}
	return defaults, nil
}

// ABOUT - Splits a "Name=value" flag and resolves the field it names
func parseFieldArg(settings []customFieldSetting, arg string) (customField, string, error) {
	name, raw, ok := strings.Cut(arg, "=")
	if !ok {
		return customField{}, "", fmt.Errorf("--field %q should look like \"Name=value\"", arg)
	}
	field, found := findCustomField(settings, strings.TrimSpace(name))
	if !found {
		return customField{}, "", fmt.Errorf("project has no custom field named %q", name)
	}
	return field, strings.TrimSpace(raw), nil
}

// ABOUT - Validates a default and stores enum choices by name, since list numbers can shift.
// Dates are kept as typed so "today" is resolved when each task is created.
func storableFieldValue(field customField, raw string) (string, error) {
	if _, err := parseCustomFieldValue(field, raw); err != nil {
		return "", err
	}
	switch field.ResourceSubtype {
	case "enum":
		option, _ := matchEnumOption(field, raw)
		return option.Name, nil
	case "multi_enum":
		var names []string
		for _, part := range strings.Split(raw, ",") {
			option, _ := matchEnumOption(field, strings.TrimSpace(part))
			names = append(names, option.Name)
		}
		return strings.Join(names, ", "), nil
	case "date":
		return strings.ToLower(raw), nil
	}
	return raw, nil
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
