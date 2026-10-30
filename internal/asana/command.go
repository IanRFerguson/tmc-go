/*
Copyright © 2024 IAN R FERGUSON IANFERGUSONRVA@gmail.com
*/
package asana

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Command is the `tmc asana` command group
var Command = &cobra.Command{
	Use:   "asana",
	Short: "Create Asana tasks from the command line",
	Long: `ASANA CLI

Run ` + "`tmc asana setup`" + ` once to store your API key and default project in
~/.tmc/asana/profile.json, then create tasks with ` + "`tmc asana create`" + `.`,
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Store an Asana API key, default project and custom field defaults",
	Long: `Creates (or overwrites) your Asana profile at ~/.tmc/asana/profile.json.

Any values not passed as flags are prompted for, including default values for
the project's custom fields. Defaults are applied to every task created in the
default project; date defaults like "today" are resolved when the task is
created. Generate a personal access token at https://app.asana.com/0/my-apps`,
	Example: `  tmc asana setup
  tmc asana setup --project "Ian's Tasks" --field "Created at=today" --field "Priority=Medium"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()
		apiKey, _ := flags.GetString("api-key")
		projectName, _ := flags.GetString("project")
		workspaceFilter, _ := flags.GetString("workspace")
		fieldArgs, _ := flags.GetStringArray("field")
		noFields, _ := flags.GetBool("no-fields")

		reader := bufio.NewReader(os.Stdin)
		existing, _ := loadProfile()

		if apiKey == "" {
			prompt := "Asana personal access token: "
			if existing.APIKey != "" {
				prompt = "Asana personal access token (leave blank to keep current): "
			}
			key, err := promptSecret(reader, prompt)
			if err != nil {
				return err
			}
			apiKey = key
			if apiKey == "" {
				apiKey = existing.APIKey
			}
		}
		if apiKey == "" {
			return fmt.Errorf("an API key is required")
		}

		client := newAPIClient(apiKey)
		user, err := client.me()
		if err != nil {
			return fmt.Errorf("could not authenticate with Asana: %w", err)
		}
		fmt.Printf("Authenticated as %s\n", user.Name)

		if projectName == "" {
			prompt := "Default project name: "
			if existing.DefaultProjectName != "" {
				prompt = fmt.Sprintf("Default project name [%s]: ", existing.DefaultProjectName)
			}
			projectName, err = promptLine(reader, prompt)
			if err != nil {
				return err
			}
			if projectName == "" {
				projectName = existing.DefaultProjectName
			}
		}
		if projectName == "" {
			return fmt.Errorf("a default project is required")
		}

		workspace, project, err := findProjectAcrossWorkspaces(client, user.Workspaces, workspaceFilter, projectName)
		if err != nil {
			return err
		}

		// Existing defaults only carry over if the default project hasn't changed
		var existingDefaults map[string]fieldDefault
		if existing.DefaultProjectGID == project.GID {
			existingDefaults = existing.FieldDefaults
		}
		fieldDefaults, err := collectFieldDefaults(client, project.GID, existingDefaults, fieldArgs, !noFields && stdinIsTerminal())
		if err != nil {
			return err
		}

		path, err := saveProfile(storedProfile{
			APIKey:             apiKey,
			WorkspaceGID:       workspace.GID,
			WorkspaceName:      workspace.Name,
			DefaultProjectGID:  project.GID,
			DefaultProjectName: project.Name,
			FieldDefaults:      fieldDefaults,
		})
		if err != nil {
			return err
		}

		fmt.Printf("✅ - Saved profile to %s (default project: %s / %s)\n", path, workspace.Name, project.Name)
		names := make([]string, 0, len(fieldDefaults))
		for name := range fieldDefaults {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Printf("  %s = %s\n", name, fieldDefaults[name])
		}
		return nil
	},
}

var createCmd = &cobra.Command{
	Use:   "create <title>",
	Short: "Create a new Asana task",
	Long: `Creates a task in your default project, or the one passed with --project.

If the project has custom fields you'll be prompted for each one; leave a
prompt blank to skip it. Use --field "Name=value" to set fields without
prompting, or --no-fields to skip the prompts entirely.

Field defaults saved by ` + "`tmc asana setup`" + ` are applied in the default project and
aren't prompted for; override one with --field, or clear it with --field "Name=".`,
	Example: `  tmc asana create "Fix the sync for ACME"
  tmc asana create Fix the sync for ACME -d "Mobilize creds expired" --due tomorrow --assign-me
  tmc asana create "Quarterly review" -p "Ops" --field "Priority=High" --no-fields
  git log -5 --oneline | tmc asana create "Review recent commits" -d -`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := cmd.Flags()
		description, _ := flags.GetString("description")
		projectName, _ := flags.GetString("project")
		due, _ := flags.GetString("due")
		assignMe, _ := flags.GetBool("assign-me")
		fieldArgs, _ := flags.GetStringArray("field")
		noFields, _ := flags.GetBool("no-fields")

		profile, err := loadProfile()
		if err != nil {
			return err
		}
		client := newAPIClient(profile.APIKey)

		if description == "-" {
			raw, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			description = strings.TrimRight(string(raw), "\n")
		}

		dueOn, err := parseDueDate(due)
		if err != nil {
			return err
		}

		project := resource{GID: profile.DefaultProjectGID, Name: profile.DefaultProjectName}
		if projectName != "" {
			project, err = findProjectInWorkspace(client, profile.WorkspaceGID, projectName)
			if err != nil {
				return err
			}
		}

		var fieldDefaults map[string]fieldDefault
		if project.GID == profile.DefaultProjectGID {
			fieldDefaults = profile.FieldDefaults
		}

		customFields, err := collectCustomFields(client, project.GID, fieldDefaults, fieldArgs, !noFields && stdinIsTerminal())
		if err != nil {
			return err
		}

		task := newTask{
			Name:         strings.Join(args, " "),
			Notes:        description,
			Projects:     []string{project.GID},
			DueOn:        dueOn,
			CustomFields: customFields,
		}
		if assignMe {
			task.Assignee = "me"
		}

		created, err := client.createTask(task)
		if err != nil {
			return err
		}

		fmt.Printf("✅ - Created \"%s\" in %s\n%s\n", created.Name, project.Name, created.PermalinkURL)
		return nil
	},
}

// ABOUT - Searches each (optionally filtered) workspace for a project, requiring exactly one match
func findProjectAcrossWorkspaces(client *apiClient, workspaces []resource, workspaceFilter string, projectName string) (resource, resource, error) {
	type match struct{ workspace, project resource }
	var matches []match

	for _, workspace := range workspaces {
		if workspaceFilter != "" && workspace.GID != workspaceFilter && !strings.EqualFold(workspace.Name, workspaceFilter) {
			continue
		}
		projects, err := client.findProjects(workspace.GID, projectName)
		if err != nil {
			return resource{}, resource{}, err
		}
		for _, project := range projects {
			matches = append(matches, match{workspace, project})
		}
	}

	switch len(matches) {
	case 0:
		return resource{}, resource{}, fmt.Errorf("no project named %q found", projectName)
	case 1:
		return matches[0].workspace, matches[0].project, nil
	default:
		lines := make([]string, len(matches))
		for i, m := range matches {
			lines[i] = fmt.Sprintf("  %s / %s (%s)", m.workspace.Name, m.project.Name, m.project.GID)
		}
		return resource{}, resource{}, fmt.Errorf(
			"%d projects match %q, narrow it down with --workspace:\n%s",
			len(matches), projectName, strings.Join(lines, "\n"),
		)
	}
}

// ABOUT - Resolves a project name (or numeric GID) within the profile's workspace
func findProjectInWorkspace(client *apiClient, workspaceGID string, projectName string) (resource, error) {
	if _, err := strconv.ParseUint(projectName, 10, 64); err == nil {
		return client.getProject(projectName)
	}

	projects, err := client.findProjects(workspaceGID, projectName)
	if err != nil {
		return resource{}, err
	}

	switch len(projects) {
	case 0:
		return resource{}, fmt.Errorf("no project named %q found in your workspace", projectName)
	case 1:
		return projects[0], nil
	default:
		gids := make([]string, len(projects))
		for i, p := range projects {
			gids[i] = p.GID
		}
		return resource{}, fmt.Errorf(
			"%d projects are named %q, pass one of these GIDs to --project instead: %s",
			len(projects), projectName, strings.Join(gids, ", "),
		)
	}
}

func init() {
	Command.AddCommand(setupCmd, createCmd)
	setupCmd.SilenceUsage = true
	createCmd.SilenceUsage = true

	setupCmd.Flags().String("api-key", "", "Asana personal access token (prompted for if omitted)")
	setupCmd.Flags().String("project", "", "Default project name to create tasks in (prompted for if omitted)")
	setupCmd.Flags().String("workspace", "", "Workspace name or GID, only needed if the project name exists in several workspaces")
	setupCmd.Flags().StringArray("field", nil, "Set a custom field default as \"Name=value\" (repeatable; empty value clears it)")
	setupCmd.Flags().Bool("no-fields", false, "Don't prompt for custom field defaults")

	createCmd.Flags().StringP("description", "d", "", "Task description; pass - to read it from stdin")
	createCmd.Flags().StringP("project", "p", "", "Project name or GID (defaults to the project in your profile)")
	createCmd.Flags().String("due", "", "Due date: YYYY-MM-DD, today or tomorrow")
	createCmd.Flags().Bool("assign-me", false, "Assign the task to yourself")
	createCmd.Flags().StringArray("field", nil, "Set a custom field as \"Name=value\" (repeatable)")
	createCmd.Flags().Bool("no-fields", false, "Don't prompt for custom fields")
}
