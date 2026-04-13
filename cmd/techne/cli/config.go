package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Edcko/techne-code/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "Initialize, show, and manage Techne Code configuration.",
	}

	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())

	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var (
		provider string
		apiKey   string
		model    string
		path     string
		force    bool
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new configuration file",
		Long:  "Create a new techne.json configuration file with provider settings.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if provider == "" {
				provider = "anthropic"
			}

			defaults := config.DefaultConfig()

			if model == "" {
				switch provider {
				case "anthropic":
					model = defaults.DefaultModel
				case "openai":
					model = "gpt-4"
				case "ollama":
					model = "llama3"
				default:
					model = defaults.DefaultModel
				}
			}

			var envRef string
			if apiKey != "" {
				envRef = apiKey
			} else if provider != "ollama" {
				envRef = "${TECHNE_" + strings.ToUpper(provider) + "_API_KEY}"
			}

			toolsEnabled := provider != "ollama"

			cfgData := configFile{
				DefaultProvider: provider,
				DefaultModel:    model,
				Providers: map[string]providerEntry{
					provider: {
						Type:         provider,
						APIKey:       envRef,
						ToolsEnabled: toolsEnabled,
					},
				},
			}

			if provider == "ollama" {
				entry := cfgData.Providers[provider]
				entry.BaseURL = "http://localhost:11434/v1"
				cfgData.Providers[provider] = entry
			}

			configPath := path
			if configPath == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get working directory: %w", err)
				}
				techneDir := filepath.Join(cwd, ".techne")
				configPath = filepath.Join(techneDir, "techne.json")
			}

			if _, err := os.Stat(configPath); err == nil && !force {
				return fmt.Errorf("config file already exists at %s (use --force to overwrite)", configPath)
			}

			dir := filepath.Dir(configPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}

			data, err := json.MarshalIndent(cfgData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			if err := os.WriteFile(configPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write config file: %w", err)
			}

			cmd.Printf("Config file created at %s\n", configPath)
			cmd.Printf("  Provider: %s\n", provider)
			cmd.Printf("  Model:    %s\n", model)
			return nil
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "LLM provider (anthropic, openai, ollama)")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key or env var reference (e.g. ${MY_KEY})")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Default model for the provider")
	cmd.Flags().StringVarP(&path, "path", "", "", "Path to write config file (default: .techne/techne.json)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config file")

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display effective configuration",
		Long:  "Show the merged configuration from all layers (defaults, files, env vars). API keys are masked.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig(cmd)

			out := showConfig{
				DefaultProvider: cfg.DefaultProvider,
				DefaultModel:    cfg.DefaultModel,
				Permissions:     cfg.Permissions.Mode,
				Providers:       make(map[string]showProvider),
			}

			for name, p := range cfg.Providers {
				sp := showProvider{
					Type:         p.Type,
					APIKey:       maskKey(p.APIKey),
					BaseURL:      p.BaseURL,
					ToolsEnabled: p.GetToolsEnabled(),
				}
				out.Providers[name] = sp
			}

			data, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			cmd.Println(string(data))
			return nil
		},
	}
}

func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

type configFile struct {
	DefaultProvider string                   `json:"default_provider"`
	DefaultModel    string                   `json:"default_model"`
	Providers       map[string]providerEntry `json:"providers,omitempty"`
}

type providerEntry struct {
	Type         string `json:"type"`
	APIKey       string `json:"api_key,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
	ToolsEnabled bool   `json:"tools_enabled"`
}

type showConfig struct {
	DefaultProvider string                  `json:"default_provider"`
	DefaultModel    string                  `json:"default_model"`
	Permissions     string                  `json:"permissions_mode"`
	Providers       map[string]showProvider `json:"providers"`
}

type showProvider struct {
	Type         string `json:"type"`
	APIKey       string `json:"api_key"`
	BaseURL      string `json:"base_url,omitempty"`
	ToolsEnabled bool   `json:"tools_enabled"`
}
