package llm

import (
	"forgor/internal/history"
	"forgor/internal/utils"
)

// BuildContextFromSystem creates an enhanced Context using system detection
func BuildContextFromSystem() Context {
	// Note: Timing is handled by caller in cmd/root.go
	return BuildContextFromSystemWithTiming(false)
}

// BuildContextFromSystemWithTiming creates an enhanced Context with optional detailed timing
func BuildContextFromSystemWithTiming(verbose bool) Context {
	var timer *utils.Timer
	if verbose {
		timer = utils.NewTimer("Context Building", verbose)
		defer timer.PrintSummary()
	}

	// Get system context with timing
	var systemCtx *utils.SystemContext
	if verbose && timer != nil {
		systemStep := timer.StartStep("System Detection")
		systemCtx = utils.GetSystemContext()
		systemStep.End()
	} else {
		systemCtx = utils.GetSystemContext()
	}

	// Build base context
	var contextStep *utils.StepTimer
	if verbose && timer != nil {
		contextStep = timer.StartStep("Context Assembly")
	}

	context := Context{
		Shell:            systemCtx.Shell,
		OS:               systemCtx.OS,
		Architecture:     systemCtx.Architecture,
		WorkingDirectory: systemCtx.WorkingDirectory,
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

	if contextStep != nil {
		contextStep.End()
	}

	return context
}

// EnhanceContextWithHistory adds command history to the context
func EnhanceContextWithHistory(context Context, historyEntries []history.HistoryEntry) Context {
	context.History = historyEntries
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
