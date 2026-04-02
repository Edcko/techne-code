package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Edcko/techne-code/internal/skills"
	"github.com/Edcko/techne-code/internal/skills/builtin"
	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/spf13/cobra"
)

func newSkillsCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage AI agent skills",
		Long:  "List, enable, and manage skills that provide specialized instructions to the AI agent.",
	}

	cmd.AddCommand(newSkillsListCmd(ctx))

	return cmd
}

func newSkillsListCmd(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		Long:  "List all available skills with their status (enabled/disabled) and triggers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig(cmd)
			registry := skills.NewRegistry()

			if err := builtin.RegisterAll(registry); err != nil {
				return fmt.Errorf("register builtin skills: %w", err)
			}

			userPath := cfg.Skills.UserSkillsPath
			if userPath == "" {
				homeDir, _ := os.UserHomeDir()
				userPath = filepath.Join(homeDir, ".config", "techne", "skills")
			}
			if _, err := os.Stat(userPath); err == nil {
				if err := registry.LoadFromPath(userPath, skill.SourceUser); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to load user skills: %v\n", err)
				}
			}

			projectPath := cfg.Skills.ProjectSkillsPath
			if projectPath == "" {
				cwd, _ := os.Getwd()
				projectPath = filepath.Join(cwd, ".techne", "skills")
			}
			if _, err := os.Stat(projectPath); err == nil {
				if err := registry.LoadFromPath(projectPath, skill.SourceProject); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to load project skills: %v\n", err)
				}
			}

			for _, name := range cfg.Skills.Disabled {
				registry.Disable(name)
			}

			skillList := registry.List()
			if len(skillList) == 0 {
				fmt.Println("No skills available.")
				return nil
			}

			fmt.Println("Available Skills:")
			fmt.Println()

			for _, s := range skillList {
				status := "enabled"
				if !registry.IsEnabled(s.Name) {
					status = "disabled"
				}

				source := string(s.Source)
				fmt.Printf("  %s (%s, %s)\n", s.Name, status, source)
				if s.Description != "" {
					fmt.Printf("    %s\n", s.Description)
				}
				if len(s.Triggers) > 0 {
					var triggerStrs []string
					for _, t := range s.Triggers {
						switch t.Type {
						case skill.TriggerAlways:
							triggerStrs = append(triggerStrs, "always")
						case skill.TriggerFilePattern:
							triggerStrs = append(triggerStrs, fmt.Sprintf("file:%s", t.Pattern))
						case skill.TriggerCommand:
							triggerStrs = append(triggerStrs, fmt.Sprintf("cmd:%s", t.Pattern))
						}
					}
					fmt.Printf("    Triggers: %s\n", strings.Join(triggerStrs, ", "))
				}
				fmt.Println()
			}

			return nil
		},
	}
}
