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
	"github.com/Edcko/techne-code/internal/llm/providers/gemini"
	"github.com/Edcko/techne-code/internal/llm/providers/ollama"
	"github.com/Edcko/techne-code/internal/llm/providers/openai"
	"github.com/Edcko/techne-code/internal/permission"
	"github.com/Edcko/techne-code/internal/skills"
	"github.com/Edcko/techne-code/internal/skills/builtin"
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

			prompt, _ := cmd.Flags().GetString("prompt")
			noTools, _ := cmd.Flags().GetBool("no-tools")
			sessionID, _ := cmd.Flags().GetString("session")
			newSession, _ := cmd.Flags().GetBool("new-session")

			if sessionID != "" && newSession {
				return fmt.Errorf("cannot use both --session and --new-session flags")
			}

			if prompt != "" {
				return runNonInteractive(ctx, cfg, prompt, noTools, sessionID)
			}

			return runInteractive(ctx, cfg, noTools, sessionID)
		},
	}

	cmd.Flags().StringP("prompt", "p", "", "Non-interactive: send a prompt and exit")
	cmd.Flags().String("session", "", "Resume a specific session")
	cmd.Flags().Bool("new-session", false, "Force create a new session")
	cmd.Flags().Bool("no-tools", false, "Disable tool use (chat-only mode)")

	return cmd
}

func validateAPIKey(apiKey string, providerType string, providerName string) error {
	if apiKey == "" && providerType != "ollama" {
		envKey := "TECHNE_" + strings.ToUpper(providerName) + "_API_KEY"
		return fmt.Errorf("API key not found for provider %q. Set %s env var", providerName, envKey)
	}
	return nil
}

func runInteractive(ctx context.Context, cfg *config.Config, noTools bool, sessionID string) error {
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

	providerCfg, ok := cfg.Providers[cfg.DefaultProvider]
	if !ok {
		return fmt.Errorf("provider %q not found in config", cfg.DefaultProvider)
	}

	apiKey := providerCfg.APIKey
	envKey := "TECHNE_" + strings.ToUpper(cfg.DefaultProvider) + "_API_KEY"
	if apiKey == "" {
		apiKey = os.Getenv(envKey)
	}

	if err := validateAPIKey(apiKey, providerCfg.Type, cfg.DefaultProvider); err != nil {
		return err
	}

	var prov provider.Provider
	switch providerCfg.Type {
	case "anthropic":
		prov = anthropic.New(apiKey)
	case "openai":
		prov = openai.NewAdapter(apiKey, providerCfg.BaseURL, nil)
	case "gemini":
		prov = gemini.NewAdapter(apiKey, providerCfg.BaseURL, nil)
	case "ollama":
		prov = ollama.New(providerCfg.BaseURL)
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
	registry.Register(&tools.ListDirTool{})
	registry.Register(tools.NewWebFetchTool())
	registry.Register(tools.NewGitTool())

	registry.Register(tools.NewSubAgentTool(tools.NewResearcherConfig(cfg.DefaultModel), prov, store, registry))
	registry.Register(tools.NewSubAgentTool(tools.NewCoderConfig(cfg.DefaultModel), prov, store, registry))
	registry.Register(tools.NewSubAgentTool(tools.NewReviewerConfig(cfg.DefaultModel), prov, store, registry))
	registry.Register(tools.NewSubAgentTool(tools.NewTesterConfig(cfg.DefaultModel), prov, store, registry))

	delegateConfigs := map[string]agent.SubAgentConfig{
		"researcher": tools.NewResearcherConfig(cfg.DefaultModel),
		"coder":      tools.NewCoderConfig(cfg.DefaultModel),
		"reviewer":   tools.NewReviewerConfig(cfg.DefaultModel),
		"tester":     tools.NewTesterConfig(cfg.DefaultModel),
	}
	registry.Register(tools.NewDelegateTool(prov, store, registry, delegateConfigs))

	skillRegistry := skills.NewRegistry()
	_ = builtin.RegisterAll(skillRegistry)

	perm := permission.NewService(
		permission.Mode(cfg.Permissions.Mode),
		cfg.Permissions.AllowedTools,
	)

	toolsEnabled := !noTools && providerCfg.GetToolsEnabled()

	model := tui.NewModel(cfg, client, store, registry, perm, bus, skillRegistry, toolsEnabled, sessionID)
	program := tea.NewProgram(model)
	model.SetProgram(program)

	if _, err := program.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	_ = toolsEnabled
	return nil
}

func runNonInteractive(ctx context.Context, cfg *config.Config, prompt string, noTools bool, sessionID string) error {
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

	if err := validateAPIKey(apiKey, providerCfg.Type, cfg.DefaultProvider); err != nil {
		return err
	}

	var prov provider.Provider
	switch providerCfg.Type {
	case "anthropic":
		prov = anthropic.New(apiKey)
	case "openai":
		prov = openai.NewAdapter(apiKey, providerCfg.BaseURL, nil)
	case "gemini":
		prov = gemini.NewAdapter(apiKey, providerCfg.BaseURL, nil)
	case "ollama":
		prov = ollama.New(providerCfg.BaseURL)
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
	registry.Register(&tools.ListDirTool{})
	registry.Register(tools.NewWebFetchTool())
	registry.Register(tools.NewGitTool())

	registry.Register(tools.NewSubAgentTool(tools.NewResearcherConfig(cfg.DefaultModel), prov, store, registry))
	registry.Register(tools.NewSubAgentTool(tools.NewCoderConfig(cfg.DefaultModel), prov, store, registry))
	registry.Register(tools.NewSubAgentTool(tools.NewReviewerConfig(cfg.DefaultModel), prov, store, registry))
	registry.Register(tools.NewSubAgentTool(tools.NewTesterConfig(cfg.DefaultModel), prov, store, registry))

	delegateConfigs := map[string]agent.SubAgentConfig{
		"researcher": tools.NewResearcherConfig(cfg.DefaultModel),
		"coder":      tools.NewCoderConfig(cfg.DefaultModel),
		"reviewer":   tools.NewReviewerConfig(cfg.DefaultModel),
		"tester":     tools.NewTesterConfig(cfg.DefaultModel),
	}
	registry.Register(tools.NewDelegateTool(prov, store, registry, delegateConfigs))

	skillRegistry := skills.NewRegistry()
	_ = builtin.RegisterAll(skillRegistry)

	perm := permission.NewService(permission.ModeAutoAllow, nil)

	ag := agent.New(client, store, registry, perm, bus)
	ag.WithSkills(skillRegistry)

	var sess *session.Session
	if sessionID != "" {
		existing, err := store.GetSession(sessionID)
		if err != nil {
			return fmt.Errorf("failed to load session %s: %w", sessionID, err)
		}
		if existing == nil {
			return fmt.Errorf("session %s not found", sessionID)
		}
		sess = existing
	} else {
		sess = &session.Session{
			Title:    "Non-interactive",
			Model:    cfg.DefaultModel,
			Provider: cfg.DefaultProvider,
		}
		if err := store.CreateSession(sess); err != nil {
			return err
		}
	}

	bus.Subscribe(func(e event.Event) {
		switch e.Type {
		case event.EventMessageDelta:
			if data, ok := e.Data.(event.ThinkingDeltaData); ok {
				_ = data
				return
			}
			if data, ok := e.Data.(event.MessageDeltaData); ok {
				fmt.Print(data.Text)
			}
		case event.EventError:
			if data, ok := e.Data.(event.ErrorData); ok {
				fmt.Fprintf(os.Stderr, "Error: %s\n", data.Message)
			}
		}
	})

	toolsEnabled := !noTools && providerCfg.GetToolsEnabled()

	if err := ag.Run(ctx, sess.ID, prompt, agent.Config{
		Model:        cfg.DefaultModel,
		MaxTokens:    4096,
		SystemPrompt: "You are Techne Code, an expert AI coding assistant. Be concise.",
		ToolsEnabled: toolsEnabled,
	}); err != nil {
		return err
	}

	fmt.Println()
	return nil
}
