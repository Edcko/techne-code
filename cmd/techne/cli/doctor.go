package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Edcko/techne-code/internal/config"
	"github.com/spf13/cobra"
)

type CheckResult struct {
	Name    string
	Pass    bool
	Message string
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose setup issues",
		Long:  "Check configuration, API keys, provider connectivity, and environment setup.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig(cmd)
			results := runDoctorChecks(cfg, checkOllamaConnectivity)

			allPass := true
			for _, r := range results {
				if !r.Pass {
					allPass = false
				}
				printResult(cmd, r)
			}

			cmd.Println()
			if allPass {
				cmd.Println("All checks passed!")
				return nil
			}
			cmd.Println("Some checks failed. Fix the issues above and try again.")
			return fmt.Errorf("doctor checks failed")
		},
	}
}

func runDoctorChecks(cfg *config.Config, ollamaChecker func(string) CheckResult) []CheckResult {
	var results []CheckResult

	results = append(results, checkConfigFile())
	results = append(results, checkDefaultProvider(cfg))
	results = append(results, checkDefaultModel(cfg))
	results = append(results, checkProviderAPIKeys(cfg)...)
	results = append(results, checkOllamaProviders(cfg, ollamaChecker)...)
	results = append(results, checkDataDirectory(cfg))
	results = append(results, checkGoVersion())

	return results
}

func checkConfigFile() CheckResult {
	paths := []string{}

	cwd, err := os.Getwd()
	if err == nil {
		projectPath := filepath.Join(cwd, ".techne", "techne.json")
		paths = append(paths, projectPath)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".config", "techne", "techne.json")
		paths = append(paths, globalPath)
	}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if !json.Valid(data) {
			return CheckResult{
				Name:    "Config file",
				Pass:    false,
				Message: fmt.Sprintf("Found %s but it contains invalid JSON", p),
			}
		}
		return CheckResult{
			Name:    "Config file",
			Pass:    true,
			Message: fmt.Sprintf("Found valid config at %s", p),
		}
	}

	return CheckResult{
		Name:    "Config file",
		Pass:    false,
		Message: "No config file found (checked .techne/techne.json and ~/.config/techne/techne.json)",
	}
}

func checkDefaultProvider(cfg *config.Config) CheckResult {
	if cfg.DefaultProvider == "" {
		return CheckResult{
			Name:    "Default provider",
			Pass:    false,
			Message: "No default provider configured",
		}
	}
	return CheckResult{
		Name:    "Default provider",
		Pass:    true,
		Message: fmt.Sprintf("Default provider: %s", cfg.DefaultProvider),
	}
}

func checkDefaultModel(cfg *config.Config) CheckResult {
	if cfg.DefaultModel == "" {
		return CheckResult{
			Name:    "Default model",
			Pass:    false,
			Message: "No default model configured",
		}
	}
	return CheckResult{
		Name:    "Default model",
		Pass:    true,
		Message: fmt.Sprintf("Default model: %s", cfg.DefaultModel),
	}
}

func checkProviderAPIKeys(cfg *config.Config) []CheckResult {
	var results []CheckResult

	if len(cfg.Providers) == 0 {
		return results
	}

	for name, pc := range cfg.Providers {
		if pc.Type == "ollama" {
			continue
		}

		apiKey := pc.APIKey
		if apiKey == "" {
			envKey := "TECHNE_" + strings.ToUpper(name) + "_API_KEY"
			apiKey = os.Getenv(envKey)
		}

		if apiKey == "" {
			envKey := "TECHNE_" + strings.ToUpper(name) + "_API_KEY"
			results = append(results, CheckResult{
				Name:    fmt.Sprintf("API key [%s]", name),
				Pass:    false,
				Message: fmt.Sprintf("No API key found for provider %q. Set %s env var or add api_key to config", name, envKey),
			})
		} else {
			masked := maskKey(apiKey)
			results = append(results, CheckResult{
				Name:    fmt.Sprintf("API key [%s]", name),
				Pass:    true,
				Message: fmt.Sprintf("API key present for %s (%s)", name, masked),
			})
		}
	}

	return results
}

func checkOllamaProviders(cfg *config.Config, checker func(string) CheckResult) []CheckResult {
	var results []CheckResult

	for name, pc := range cfg.Providers {
		if pc.Type != "ollama" {
			continue
		}

		baseURL := pc.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		baseURL = strings.TrimSuffix(baseURL, "/")
		baseURL = strings.TrimSuffix(baseURL, "/v1")

		result := checker(baseURL)
		result.Name = fmt.Sprintf("Ollama connectivity [%s]", name)
		results = append(results, result)
	}

	return results
}

func checkOllamaConnectivity(baseURL string) CheckResult {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(baseURL + "/api/tags")
	if err != nil {
		return CheckResult{
			Pass:    false,
			Message: fmt.Sprintf("Cannot reach Ollama at %s (%v)", baseURL, err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return CheckResult{
			Pass:    false,
			Message: fmt.Sprintf("Ollama at %s returned status %d", baseURL, resp.StatusCode),
		}
	}

	return CheckResult{
		Pass:    true,
		Message: fmt.Sprintf("Ollama reachable at %s", baseURL),
	}
}

func checkDataDirectory(cfg *config.Config) CheckResult {
	dataDir := cfg.Options.DataDirectory
	if dataDir == "" {
		dataDir = ".techne"
	}

	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		return CheckResult{
			Name:    "Data directory",
			Pass:    false,
			Message: fmt.Sprintf("Cannot create data directory %q: %v", dataDir, err),
		}
	}

	tmpFile := filepath.Join(dataDir, ".doctor_write_test")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		return CheckResult{
			Name:    "Data directory",
			Pass:    false,
			Message: fmt.Sprintf("Data directory %q exists but is not writable: %v", dataDir, err),
		}
	}
	os.Remove(tmpFile)

	absPath, _ := filepath.Abs(dataDir)
	return CheckResult{
		Name:    "Data directory",
		Pass:    true,
		Message: fmt.Sprintf("Data directory writable at %s", absPath),
	}
}

func checkGoVersion() CheckResult {
	return CheckResult{
		Name:    "Go version",
		Pass:    true,
		Message: fmt.Sprintf("Go %s (%s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}
}

func printResult(cmd *cobra.Command, r CheckResult) {
	icon := "✅"
	if !r.Pass {
		icon = "❌"
	}
	cmd.Printf("  %s %s: %s\n", icon, r.Name, r.Message)
}
