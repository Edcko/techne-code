package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/Edcko/techne-code/internal/agent"
	"github.com/Edcko/techne-code/internal/config"
	"github.com/Edcko/techne-code/internal/db"
	eventbus "github.com/Edcko/techne-code/internal/event"
	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/internal/llm/providers/anthropic"
	"github.com/Edcko/techne-code/internal/llm/providers/openai"
	"github.com/Edcko/techne-code/internal/permission"
	"github.com/Edcko/techne-code/internal/tools"
	"github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
	"github.com/Edcko/techne-code/tui"
)

func newChatCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Start interactive chat session",
		Long:  "Start an interactive TUI chat session with the AI coding agent.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig(cmd)

			// Check for non-interactive mode
			prompt, _ := cmd.Flags().GetString("prompt")
			if prompt != "" {
				return runNonInteractive(ctx, cfg, prompt)
			}

			return runInteractive(ctx, cfg)
		},
	}

	cmd.Flags().StringP("prompt", "p", "", "Non-interactive: send a prompt and exit")
	cmd.Flags().String("session", "", "Resume a specific session")
	cmd.Flags().Bool("new-session", false, "Force create a new session")

	return cmd
}

func runInteractive(ctx context.Context, cfg *config.Config) error {
	// 1. Setup database
	dataDir := cfg.Options.DataDirectory
	if dataDir == "" {
		dataDir = ".techne"
	}
	os.MkdirAll(dataDir, 0755)

	database, err := db.Open(dataDir + "/techne.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	store := db.NewSessionStore(database)

	// 2. Setup event bus
	bus := eventbus.NewChannelEventBus()

	// 3. Setup provider
	providerCfg, ok := cfg.Providers[cfg.DefaultProvider]
	if !ok {
		return fmt.Errorf("provider %q not found in config", cfg.DefaultProvider)
	}

	apiKey := providerCfg.APIKey
	envKey := "TECHNE_" + strings.ToUpper(cfg.DefaultProvider) + "_API_KEY"
	if apiKey == "" {
		apiKey = os.Getenv(envKey)
	}
	if apiKey == "" {
		return fmt.Errorf("API key not found for provider %q. Set %s env var", cfg.DefaultProvider, envKey)
	}

	var prov provider.Provider
	switch providerCfg.Type {
	case "anthropic":
		prov = anthropic.New(apiKey)
	case "openai":
		prov = openai.NewAdapter(apiKey, providerCfg.BaseURL)
	default:
		return fmt.Errorf("unsupported provider type: %q", providerCfg.Type)
	}

	// 4. Setup LLM client
	client := llm.NewClient(prov, bus)

	// 5. Setup tool registry
	registry := tools.NewRegistry()
	registry.Register(&tools.ReadFileTool{})
	registry.Register(&tools.WriteFileTool{})
	registry.Register(&tools.EditFileTool{})
	registry.Register(tools.NewBashTool())
	registry.Register(&tools.GrepTool{})
	registry.Register(&tools.GlobTool{})

	// 6. Setup permissions
	perm := permission.NewService(
		permission.Mode(cfg.Permissions.Mode),
		cfg.Permissions.AllowedTools,
	)

	// 7. Create and run TUI
	model := tui.NewModel(cfg, client, store, registry, perm, bus)
	program := tea.NewProgram(model)
	model.SetProgram(program)

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func runNonInteractive(ctx context.Context, cfg *config.Config, prompt string) error {
	// Setup same as interactive but without TUI
	dataDir := cfg.Options.DataDirectory
	if dataDir == "" {
		dataDir = ".techne"
	}
	os.MkdirAll(dataDir, 0755)

	database, err := db.Open(dataDir + "/techne.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	store := db.NewSessionStore(database)

	bus := eventbus.NewChannelEventBus()
	defer bus.Close()

	providerCfg, ok := cfg.Providers[cfg.DefaultProvider]
	if !ok {
		return fmt.Errorf("provider %q not found", cfg.DefaultProvider)
	}

	apiKey := providerCfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("TECHNE_" + strings.ToUpper(cfg.DefaultProvider) + "_API_KEY")
	}

	var prov provider.Provider
	switch providerCfg.Type {
	case "anthropic":
		prov = anthropic.New(apiKey)
	case "openai":
		prov = openai.NewAdapter(apiKey, providerCfg.BaseURL)
	default:
		return fmt.Errorf("unsupported provider type: %q", providerCfg.Type)
	}

	client := llm.NewClient(prov, bus)
	registry := tools.NewRegistry()
	registry.Register(&tools.ReadFileTool{})
	registry.Register(&tools.WriteFileTool{})
	registry.Register(&tools.EditFileTool{})
	registry.Register(tools.NewBashTool())
	registry.Register(&tools.GrepTool{})
	registry.Register(&tools.GlobTool{})

	perm := permission.NewService(permission.ModeAutoAllow, nil)

	ag := agent.New(client, store, registry, perm, bus)

	sess := &session.Session{
		Title:    "Non-interactive",
		Model:    cfg.DefaultModel,
		Provider: cfg.DefaultProvider,
	}
	if err := store.CreateSession(sess); err != nil {
		return err
	}

	// Subscribe to events for output
	bus.Subscribe(func(e event.Event) {
		switch e.Type {
		case event.EventMessageDelta:
			if data, ok := e.Data.(event.MessageDeltaData); ok {
				fmt.Print(data.Text)
			}
		case event.EventError:
			if data, ok := e.Data.(event.ErrorData); ok {
				fmt.Fprintf(os.Stderr, "Error: %s\n", data.Message)
			}
		}
	})

	if err := ag.Run(ctx, sess.ID, prompt, agent.Config{
		Model:        cfg.DefaultModel,
		MaxTokens:    4096,
		SystemPrompt: "You are Techne Code, an expert AI coding assistant. Be concise.",
	}); err != nil {
		return err
	}

	fmt.Println()
	return nil
}
