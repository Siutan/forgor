package llm

import (
	"forgor/internal/utils"
)

// BuildContextFromSystem creates an enhanced Context using system detection
func BuildContextFromSystem() Context {
	systemCtx := utils.GetSystemContext()

	context := Context{
		OS:               systemCtx.OS,
		Shell:            systemCtx.Shell,
		WorkingDirectory: systemCtx.WorkingDirectory,
		Architecture:     systemCtx.Architecture,
		User:             systemCtx.User,
		HomeDirectory:    systemCtx.HomeDirectory,
		Environment:      systemCtx.Environment,
		ToolsAvailable:   systemCtx.Tools.Available,
		ToolsSummary:     utils.GetToolContextSummary(),
	}

	// Extract package managers
	context.PackageManagers = systemCtx.Tools.PackageManagers

	// Extract language names
	context.Languages = make([]string, len(systemCtx.Tools.Languages))
	for i, lang := range systemCtx.Tools.Languages {
		context.Languages[i] = lang.Name
	}

	// Extract development tool names
	context.DevelopmentTools = make([]string, len(systemCtx.Tools.DevelopmentTools))
	for i, tool := range systemCtx.Tools.DevelopmentTools {
		context.DevelopmentTools[i] = tool.Name
	}

	// Extract other tool categories
	context.ContainerTools = systemCtx.Tools.ContainerTools
	context.CloudTools = systemCtx.Tools.CloudTools
	context.DatabaseTools = systemCtx.Tools.DatabaseTools
	context.NetworkTools = systemCtx.Tools.NetworkTools

	return context
}

// EnhanceContextWithHistory adds command history to the context
func EnhanceContextWithHistory(context Context, history []string) Context {
	context.History = history
	return context
}

// EnhanceContextWithUserInput adds user-provided context
func EnhanceContextWithUserInput(context Context, userContext string) Context {
	context.UserContext = userContext
	return context
}

// GetToolCapabilitiesText returns a formatted text description of available tools
func GetToolCapabilitiesText(context Context) string {
	if context.ToolsSummary != "" {
		return context.ToolsSummary
	}
	return "Standard system commands available"
}

// IsToolAvailableInContext checks if a tool is available in the given context
func IsToolAvailableInContext(context Context, tool string) bool {
	if context.ToolsAvailable == nil {
		return false
	}
	available, exists := context.ToolsAvailable[tool]
	return exists && available
}
