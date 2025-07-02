package config

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

// ValidToolCategories defines the available tool categories
var ValidToolCategories = []string{
	"package_managers",
	"languages",
	"development_tools",
	"system_commands",
	"container_tools",
	"cloud_tools",
	"database_tools",
	"network_tools",
	"other",
}

// AddCustomTools adds tools to a specific category
func AddCustomTools(category string, tools []string) error {
	if !slices.Contains(ValidToolCategories, category) {
		return fmt.Errorf("invalid category '%s'. Valid categories: %s",
			category, strings.Join(ValidToolCategories, ", "))
	}

	config, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate that tools exist in PATH
	validTools := make([]string, 0, len(tools))
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool == "" {
			continue
		}

		if _, err := exec.LookPath(tool); err != nil {
			fmt.Printf("Warning: Tool '%s' not found in PATH, adding anyway\n", tool)
		}
		validTools = append(validTools, tool)
	}

	if len(validTools) == 0 {
		return fmt.Errorf("no valid tools to add")
	}

	// Add tools to the appropriate category (avoiding duplicates)
	switch category {
	case "package_managers":
		config.CustomTools.PackageManagers = addUniqueTools(config.CustomTools.PackageManagers, validTools)
	case "languages":
		config.CustomTools.Languages = addUniqueTools(config.CustomTools.Languages, validTools)
	case "development_tools":
		config.CustomTools.DevelopmentTools = addUniqueTools(config.CustomTools.DevelopmentTools, validTools)
	case "system_commands":
		config.CustomTools.SystemCommands = addUniqueTools(config.CustomTools.SystemCommands, validTools)
	case "container_tools":
		config.CustomTools.ContainerTools = addUniqueTools(config.CustomTools.ContainerTools, validTools)
	case "cloud_tools":
		config.CustomTools.CloudTools = addUniqueTools(config.CustomTools.CloudTools, validTools)
	case "database_tools":
		config.CustomTools.DatabaseTools = addUniqueTools(config.CustomTools.DatabaseTools, validTools)
	case "network_tools":
		config.CustomTools.NetworkTools = addUniqueTools(config.CustomTools.NetworkTools, validTools)
	case "other":
		config.CustomTools.Other = addUniqueTools(config.CustomTools.Other, validTools)
	}

	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Added %d tool(s) to %s: %s\n", len(validTools), category, strings.Join(validTools, ", "))
	return nil
}

// RemoveCustomTools removes tools from a specific category
func RemoveCustomTools(category string, tools []string) error {
	if !slices.Contains(ValidToolCategories, category) {
		return fmt.Errorf("invalid category '%s'. Valid categories: %s",
			category, strings.Join(ValidToolCategories, ", "))
	}

	config, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	toolsToRemove := make([]string, 0, len(tools))
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool != "" {
			toolsToRemove = append(toolsToRemove, tool)
		}
	}

	if len(toolsToRemove) == 0 {
		return fmt.Errorf("no tools specified to remove")
	}

	removedCount := 0

	// Remove tools from the appropriate category
	switch category {
	case "package_managers":
		config.CustomTools.PackageManagers, removedCount = removeTools(config.CustomTools.PackageManagers, toolsToRemove)
	case "languages":
		config.CustomTools.Languages, removedCount = removeTools(config.CustomTools.Languages, toolsToRemove)
	case "development_tools":
		config.CustomTools.DevelopmentTools, removedCount = removeTools(config.CustomTools.DevelopmentTools, toolsToRemove)
	case "system_commands":
		config.CustomTools.SystemCommands, removedCount = removeTools(config.CustomTools.SystemCommands, toolsToRemove)
	case "container_tools":
		config.CustomTools.ContainerTools, removedCount = removeTools(config.CustomTools.ContainerTools, toolsToRemove)
	case "cloud_tools":
		config.CustomTools.CloudTools, removedCount = removeTools(config.CustomTools.CloudTools, toolsToRemove)
	case "database_tools":
		config.CustomTools.DatabaseTools, removedCount = removeTools(config.CustomTools.DatabaseTools, toolsToRemove)
	case "network_tools":
		config.CustomTools.NetworkTools, removedCount = removeTools(config.CustomTools.NetworkTools, toolsToRemove)
	case "other":
		config.CustomTools.Other, removedCount = removeTools(config.CustomTools.Other, toolsToRemove)
	}

	if removedCount == 0 {
		return fmt.Errorf("no tools were found to remove from %s", category)
	}

	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed %d tool(s) from %s\n", removedCount, category)
	return nil
}

// ListCustomTools lists all custom tools or tools in a specific category
func ListCustomTools(category string) error {
	config, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if category != "" {
		if !slices.Contains(ValidToolCategories, category) {
			return fmt.Errorf("invalid category '%s'. Valid categories: %s",
				category, strings.Join(ValidToolCategories, ", "))
		}
		return listCategoryTools(config, category)
	}

	// List all categories
	fmt.Println("Custom Tools:")
	fmt.Println("=============")

	totalTools := 0
	for _, cat := range ValidToolCategories {
		tools := getCategoryTools(config, cat)
		if len(tools) > 0 {
			fmt.Printf("\n%s: %s\n", strings.ReplaceAll(cat, "_", " "), strings.Join(tools, ", "))
			totalTools += len(tools)
		}
	}

	if totalTools == 0 {
		fmt.Println("\nNo custom tools configured.")
		fmt.Println("Add tools with: forgor config tools add <category> <tool1,tool2,...>")
	} else {
		fmt.Printf("\nTotal: %d custom tools\n", totalTools)
	}

	return nil
}

// ClearCustomTools clears all tools from a category or all categories
func ClearCustomTools(category string) error {
	config, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if category == "" || category == "all" {
		// Clear all custom tools
		config.CustomTools = CustomToolsConfig{}
		fmt.Println("Cleared all custom tools")
	} else {
		if !slices.Contains(ValidToolCategories, category) {
			return fmt.Errorf("invalid category '%s'. Valid categories: %s",
				category, strings.Join(ValidToolCategories, ", "))
		}

		// Clear specific category
		switch category {
		case "package_managers":
			config.CustomTools.PackageManagers = []string{}
		case "languages":
			config.CustomTools.Languages = []string{}
		case "development_tools":
			config.CustomTools.DevelopmentTools = []string{}
		case "system_commands":
			config.CustomTools.SystemCommands = []string{}
		case "container_tools":
			config.CustomTools.ContainerTools = []string{}
		case "cloud_tools":
			config.CustomTools.CloudTools = []string{}
		case "database_tools":
			config.CustomTools.DatabaseTools = []string{}
		case "network_tools":
			config.CustomTools.NetworkTools = []string{}
		case "other":
			config.CustomTools.Other = []string{}
		}
		fmt.Printf("Cleared %s category\n", category)
	}

	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetCustomTools returns all custom tools from the config
func GetCustomTools(config *Config) *CustomToolsConfig {
	if config == nil {
		return &CustomToolsConfig{}
	}
	return &config.CustomTools
}

// Helper functions

func addUniqueTools(existing []string, newTools []string) []string {
	result := make([]string, len(existing))
	copy(result, existing)

	for _, tool := range newTools {
		if !slices.Contains(result, tool) {
			result = append(result, tool)
		}
	}

	return result
}

func removeTools(existing []string, toRemove []string) ([]string, int) {
	result := make([]string, 0, len(existing))
	removedCount := 0

	for _, tool := range existing {
		if !slices.Contains(toRemove, tool) {
			result = append(result, tool)
		} else {
			removedCount++
		}
	}

	return result, removedCount
}

func getCategoryTools(config *Config, category string) []string {
	switch category {
	case "package_managers":
		return config.CustomTools.PackageManagers
	case "languages":
		return config.CustomTools.Languages
	case "development_tools":
		return config.CustomTools.DevelopmentTools
	case "system_commands":
		return config.CustomTools.SystemCommands
	case "container_tools":
		return config.CustomTools.ContainerTools
	case "cloud_tools":
		return config.CustomTools.CloudTools
	case "database_tools":
		return config.CustomTools.DatabaseTools
	case "network_tools":
		return config.CustomTools.NetworkTools
	case "other":
		return config.CustomTools.Other
	default:
		return []string{}
	}
}

func listCategoryTools(config *Config, category string) error {
	tools := getCategoryTools(config, category)

	categoryName := strings.ReplaceAll(category, "_", " ")
	fmt.Printf("%s: ", strings.Title(categoryName))

	if len(tools) == 0 {
		fmt.Println("(none)")
	} else {
		fmt.Println(strings.Join(tools, ", "))
	}

	return nil
}
