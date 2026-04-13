package tui

import (
	"fmt"
	"strings"

	"github.com/Edcko/techne-code/internal/config"
)

type CommandType string

const (
	CommandModel    CommandType = "model"
	CommandProvider CommandType = "provider"
	CommandHelp     CommandType = "help"
	CommandClear    CommandType = "clear"
	CommandUnknown  CommandType = "unknown"
)

type CommandResult struct {
	Handled     bool
	Command     CommandType
	Message     string
	IsError     bool
	NewModel    string
	NewProvider string
	ClearChat   bool
}

func ParseSlashCommand(input string) (CommandType, string, bool) {
	if !strings.HasPrefix(input, "/") {
		return "", "", false
	}

	trimmed := strings.TrimSpace(input)
	if trimmed == "/" {
		return CommandUnknown, "", true
	}

	fields := strings.Fields(trimmed)
	name := strings.TrimPrefix(fields[0], "/")

	switch name {
	case "model":
		if len(fields) < 2 {
			return CommandModel, "", true
		}
		return CommandModel, fields[1], true
	case "provider":
		if len(fields) < 2 {
			return CommandProvider, "", true
		}
		return CommandProvider, fields[1], true
	case "help":
		return CommandHelp, "", true
	case "clear":
		return CommandClear, "", true
	default:
		return CommandUnknown, name, true
	}
}

func ExecuteSlashCommand(input string, cfg *config.Config) CommandResult {
	cmdType, arg, isCmd := ParseSlashCommand(input)
	if !isCmd {
		return CommandResult{Handled: false}
	}

	switch cmdType {
	case CommandModel:
		return executeModelSwitch(arg, cfg)
	case CommandProvider:
		return executeProviderSwitch(arg, cfg)
	case CommandHelp:
		return executeHelp()
	case CommandClear:
		return executeClear()
	case CommandUnknown:
		return executeUnknownCommand(arg)
	default:
		return executeUnknownCommand(arg)
	}
}

func executeModelSwitch(modelName string, cfg *config.Config) CommandResult {
	if modelName == "" {
		return CommandResult{
			Handled: true,
			Command: CommandModel,
			Message: "Usage: /model <name>\nAvailable models: " + formatModelList(cfg),
			IsError: true,
		}
	}

	if !isValidModel(modelName, cfg) {
		return CommandResult{
			Handled: true,
			Command: CommandModel,
			Message: fmt.Sprintf("Unknown model %q. Available: %s", modelName, formatModelList(cfg)),
			IsError: true,
		}
	}

	return CommandResult{
		Handled:  true,
		Command:  CommandModel,
		Message:  fmt.Sprintf("Switched model to %s", modelName),
		NewModel: modelName,
	}
}

func executeProviderSwitch(providerName string, cfg *config.Config) CommandResult {
	if providerName == "" {
		return CommandResult{
			Handled: true,
			Command: CommandProvider,
			Message: "Usage: /provider <name>\nAvailable providers: " + formatProviderList(cfg),
			IsError: true,
		}
	}

	if !isValidProvider(providerName, cfg) {
		return CommandResult{
			Handled: true,
			Command: CommandProvider,
			Message: fmt.Sprintf("Unknown provider %q. Available: %s", providerName, formatProviderList(cfg)),
			IsError: true,
		}
	}

	return CommandResult{
		Handled:     true,
		Command:     CommandProvider,
		Message:     fmt.Sprintf("Switched provider to %s", providerName),
		NewProvider: providerName,
	}
}

func executeHelp() CommandResult {
	help := `Available commands:
  /model <name>     - Switch the AI model for subsequent messages
  /provider <name>  - Switch the AI provider (model + API key + adapter)
  /clear            - Clear conversation history (start fresh in same session)
  /help             - Show this help message`
	return CommandResult{
		Handled: true,
		Command: CommandHelp,
		Message: help,
	}
}

func executeClear() CommandResult {
	return CommandResult{
		Handled:   true,
		Command:   CommandClear,
		Message:   "Conversation cleared. Starting fresh.",
		ClearChat: true,
	}
}

func executeUnknownCommand(name string) CommandResult {
	return CommandResult{
		Handled: true,
		Command: CommandUnknown,
		Message: fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", name),
		IsError: true,
	}
}

func isValidModel(modelName string, cfg *config.Config) bool {
	if info, ok := cfg.Providers[cfg.DefaultProvider]; ok {
		for _, m := range info.Models {
			if m == modelName {
				return true
			}
		}
	}

	for _, info := range cfg.Providers {
		for _, m := range info.Models {
			if m == modelName {
				return true
			}
		}
	}

	return false
}

func isValidProvider(providerName string, cfg *config.Config) bool {
	_, ok := cfg.Providers[providerName]
	return ok
}

func formatModelList(cfg *config.Config) string {
	if info, ok := cfg.Providers[cfg.DefaultProvider]; ok && len(info.Models) > 0 {
		return strings.Join(info.Models, ", ")
	}

	var all []string
	for _, info := range cfg.Providers {
		all = append(all, info.Models...)
	}
	return strings.Join(all, ", ")
}

func formatProviderList(cfg *config.Config) string {
	names := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
