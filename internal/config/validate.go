package config

import (
	"fmt"
	"strings"
)

// validProviderTypes is the list of valid provider types.
var validProviderTypes = map[string]bool{
	"anthropic": true,
	"openai":    true,
	"ollama":    true,
}

// Validate checks the configuration for errors and returns helpful messages.
// It validates required fields, provider types, and value constraints.
func Validate(c *Config) error {
	var errors []string

	// Validate default_provider
	if c.DefaultProvider == "" {
		errors = append(errors, "default_provider is required and cannot be empty")
	}

	// Validate default_model
	if c.DefaultModel == "" {
		errors = append(errors, "default_model is required and cannot be empty")
	}

	// Validate providers if any are defined
	for name, provider := range c.Providers {
		if !validProviderTypes[provider.Type] {
			validTypes := make([]string, 0, len(validProviderTypes))
			for t := range validProviderTypes {
				validTypes = append(validTypes, t)
			}
			errors = append(errors, fmt.Sprintf(
				"provider %q has invalid type %q (valid types: %s)",
				name, provider.Type, strings.Join(validTypes, ", "),
			))
		}
	}

	// Validate that if default_provider references a provider, it exists
	// Note: default_provider can also be a built-in provider name, so we only
	// validate if the user has defined providers and the name matches
	if len(c.Providers) > 0 {
		if _, exists := c.Providers[c.DefaultProvider]; !exists {
			// Check if it's a built-in provider type
			if !validProviderTypes[c.DefaultProvider] {
				errors = append(errors, fmt.Sprintf(
					"default_provider %q is not defined in providers and is not a valid built-in provider",
					c.DefaultProvider,
				))
			}
		}
	}

	// Validate options
	if c.Options.MaxBashTimeout <= 0 {
		errors = append(errors, "options.max_bash_timeout must be greater than 0")
	}

	if c.Options.MaxOutputChars <= 0 {
		errors = append(errors, "options.max_output_chars must be greater than 0")
	}

	// Validate permissions mode
	validModes := map[string]bool{
		"interactive": true,
		"auto_allow":  true,
		"auto_deny":   true,
	}
	if c.Permissions.Mode != "" && !validModes[c.Permissions.Mode] {
		errors = append(errors, fmt.Sprintf(
			"permissions.mode must be one of: interactive, auto_allow, auto_deny (got %q)",
			c.Permissions.Mode,
		))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}
