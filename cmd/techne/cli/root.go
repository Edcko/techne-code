// Package cli provides the command-line interface for Techne Code.
// It uses cobra for command parsing and organizes subcommands for
// chat, session management, and version information.
package cli

import (
	"context"
	"os"

	"github.com/Edcko/techne-code/internal/config"
	"github.com/spf13/cobra"
)

// Execute creates and executes the root command.
// It sets up all subcommands and global flags, then runs the command tree.
func Execute(ctx context.Context, version string) error {
	// Create chat command first so we can reference it from root
	chatCmd := newChatCmd(ctx)

	rootCmd := &cobra.Command{
		Use:           "techne",
		Short:         "Techne Code — Open source coding AI agent",
		Long:          "An extensible, multi-provider AI coding agent with sub-agent orchestration.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		// Default to chat when no subcommand is provided
		RunE: chatCmd.RunE,
	}

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().String("provider", "", "LLM provider to use")
	rootCmd.PersistentFlags().String("model", "", "LLM model to use")

	// Persistent pre-run: load config
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return loadConfig(cmd)
	}

	// Add chat-specific flags to root for when running without subcommand
	// These are passed through to the chat command
	rootCmd.Flags().StringP("prompt", "p", "", "Non-interactive: send a prompt and exit")
	rootCmd.Flags().String("session", "", "Resume a specific session")
	rootCmd.Flags().Bool("new-session", false, "Force create a new session")

	// Subcommands
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(newSessionCmd(ctx))
	rootCmd.AddCommand(newSkillsCmd(ctx))
	rootCmd.AddCommand(newVersionCmd(version))

	return rootCmd.ExecuteContext(ctx)
}

// loadConfig loads configuration and stores it in cobra context.
// It respects the priority: CLI flags > config file > defaults.
func loadConfig(cmd *cobra.Command) error {
	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Check for custom config flag
	configPath, _ := cmd.Flags().GetString("config")

	var cfg *config.Config

	if configPath != "" {
		cfg, err = config.LoadFromFile(configPath)
	} else {
		cfg, err = config.Load(cwd)
	}

	// If config loading fails, use defaults (don't crash)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Apply CLI flag overrides
	if provider, _ := cmd.Flags().GetString("provider"); provider != "" {
		cfg.DefaultProvider = provider
		if _, exists := cfg.Providers[provider]; !exists {
			cfg.Providers[provider] = config.ProviderConfig{Type: provider}
		}
	}
	if model, _ := cmd.Flags().GetString("model"); model != "" {
		cfg.DefaultModel = model
	}

	// Store in cobra context
	cmd.SetContext(context.WithValue(cmd.Context(), configKey{}, cfg))
	return nil
}

// getConfig retrieves the config from the cobra command context.
// Panics if config is not found (should never happen if loadConfig ran).
func getConfig(cmd *cobra.Command) *config.Config {
	cfg, ok := cmd.Context().Value(configKey{}).(*config.Config)
	if !ok {
		// Fallback to defaults if context is missing
		return config.DefaultConfig()
	}
	return cfg
}

// configKey is the context key for storing config.
type configKey struct{}
