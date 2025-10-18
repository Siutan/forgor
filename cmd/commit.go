package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"forgor/internal/config"
	"forgor/internal/llm"
	"forgor/internal/utils"

	"github.com/spf13/cobra"
)

var (
	commitDryRun  bool
	commitEdit    bool
	commitVerbose bool
)

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a conventional commit message from git changes",
	Long: `Generate a conventional commit message using AI based on your git changes.

The command analyzes staged files (or unstaged changes if nothing is staged) along
with any untracked files, and generates a commit message following the Conventional
Commits specification.

After generation, you can:
- [e] Edit the message in your default editor
- [a] Apply the commit (runs git commit with the message)
- [d] Delete/cancel without committing

Examples:
  forgor commit                      # Generate commit message (includes untracked files)
  forgor commit --dry-run            # Preview without committing
  forgor commit --profile anthropic  # Use specific LLM provider
  forgor commit --verbose            # Show model explanation and extra diagnostics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCommitGeneration()
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().BoolVar(&commitDryRun, "dry-run", false, "preview the commit message without prompting")
	commitCmd.Flags().BoolVar(&commitEdit, "edit", false, "automatically open editor after generation")
	commitCmd.Flags().BoolVar(&commitVerbose, "verbose", false, "show model explanation and extra diagnostics") // NEW
}

func runCommitGeneration() error {
	// Check if we're in a git repository
	if !isGitRepository() {
		return fmt.Errorf("not a git repository")
	}

	// Determine what changes to analyze
	stagedChanges, err := getGitStagedChanges()
	if err != nil {
		return fmt.Errorf("failed to get staged changes: %w", err)
	}

	unstagedChanges := ""
	untrackedFiles := ""
	useStaged := len(strings.TrimSpace(stagedChanges)) > 0

	if !useStaged {
		unstagedChanges, err = getGitUnstagedChanges()
		if err != nil {
			return fmt.Errorf("failed to get unstaged changes: %w", err)
		}
	}

	// Always check for untracked files
	untrackedFiles, _ = getUntrackedFilesContent()
	untrackedFilesList, _ := getUntrackedFiles()

	// Check if we have any changes at all
	hasChanges := len(strings.TrimSpace(stagedChanges)) > 0 ||
		len(strings.TrimSpace(unstagedChanges)) > 0 ||
		len(strings.TrimSpace(untrackedFiles)) > 0

	if !hasChanges {
		return fmt.Errorf("no changes detected (staged, unstaged, or untracked)")
	}

	// Get file statistics
	fileStats, _ := getGitFileStats(useStaged)
	gitStatus, _ := getGitStatus()

	// Show what we're analyzing
	if useStaged {
		fmt.Printf("%s Analyzing staged changes...\n", utils.Styled("→", utils.StyleInfo))
	} else {
		fmt.Printf("%s No staged files found. Analyzing unstaged changes...\n", utils.Styled("→", utils.StyleWarning))
	}

	if fileStats != "" {
		fmt.Printf("%s\n", utils.Styled(strings.TrimSpace(fileStats), utils.StyleSubtle))
	}

	// Show untracked files info
	if len(untrackedFilesList) > 0 {
		fmt.Printf("%s Found %d untracked file(s): %s\n",
			utils.Styled("→", utils.StyleInfo),
			len(untrackedFilesList),
			utils.Styled(strings.Join(untrackedFilesList, ", "), utils.StyleSubtle))
	}

	// Start spinner
	stopSpinner := startSpinner("Generating commit message")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		stopSpinner()
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create LLM factory and get provider
	factory := llm.NewFactory(cfg)
	provider, err := factory.GetProvider(profile)
	if err != nil {
		stopSpinner()
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// Build the prompt for commit message generation
	var changes string
	if useStaged {
		changes = stagedChanges
	} else {
		changes = unstagedChanges
	}

	prompt := buildCommitPrompt(changes, useStaged, fileStats, gitStatus, untrackedFiles)

	// Generate commit message
	ctx := context.Background()
	requestContext := llm.BuildContextFromSystem()

	response, err := provider.GenerateCommand(ctx, &llm.Request{
		Query:   prompt,
		Context: requestContext,
		Options: llm.RequestOptions{
			IncludeExplanation: commitVerbose,
			MaxTokens:          300,
			Temperature:        0.3,
		},
	})

	stopSpinner()

	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	// debug: print raw response
	fmt.Printf("Command response: %s\n Command explanation: %s\n", response.Command, response.Explanation)

	// Extract commit message from response
	commitMessage := extractCommitMessage(response.Command, response.Explanation, commitVerbose)

	// Validate commit message
	if strings.TrimSpace(commitMessage) == "" {
		return fmt.Errorf("failed to generate a valid commit message. Please try again or specify changes manually")
	}

	// Display the generated commit message
	fmt.Printf("\n%s\n", utils.Divider("GENERATED COMMIT MESSAGE", utils.StyleSuccess))
	fmt.Printf("%s\n", utils.Styled(commitMessage, utils.StyleCommand))
	fmt.Printf("%s\n\n", utils.Divider("", utils.StyleSuccess))

	if commitVerbose && response.Explanation != "" {
		fmt.Printf("%s %s\n\n", utils.Styled("Explanation:", utils.StyleInfo), response.Explanation)
	}

	// Dry run mode - just show and exit
	if commitDryRun {
		return nil
	}

	// Auto-edit mode
	if commitEdit {
		editedMessage, err := openEditorForCommit(commitMessage)
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}
		commitMessage = editedMessage
	}

	// Interactive prompt
	for {
		fmt.Printf("%s [e]dit / [a]pply / [d]elete: ", utils.Styled("→", utils.StyleInfo))

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		choice := strings.ToLower(strings.TrimSpace(input))

		switch choice {
		case "e", "edit":
			editedMessage, err := openEditorForCommit(commitMessage)
			if err != nil {
				fmt.Printf("%s Failed to open editor: %v\n", utils.Styled("✗", utils.StyleError), err)
				continue
			}
			commitMessage = editedMessage
			fmt.Printf("\n%s\n", utils.Divider("EDITED COMMIT MESSAGE", utils.StyleSuccess))
			fmt.Printf("%s\n", utils.Styled(commitMessage, utils.StyleCommand))
			fmt.Printf("%s\n\n", utils.Divider("", utils.StyleSuccess))

		case "a", "apply":
			if useStaged {
				err = applyCommit(commitMessage)
			} else {
				fmt.Printf("%s No files are staged. Would you like to stage all changes? [y/n]: ", utils.Styled("?", utils.StyleWarning))
				stageInput, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(stageInput)) == "y" {
					if err := stageAllChanges(); err != nil {
						return fmt.Errorf("failed to stage changes: %w", err)
					}
					err = applyCommit(commitMessage)
				} else {
					fmt.Printf("%s Commit cancelled. Please stage files manually.\n", utils.Styled("→", utils.StyleInfo))
					return nil
				}
			}

			if err != nil {
				return fmt.Errorf("failed to apply commit: %w", err)
			}
			fmt.Printf("%s Commit applied successfully!\n", utils.Styled("✓", utils.StyleSuccess))
			return nil

		case "d", "delete", "cancel":
			fmt.Printf("%s Commit cancelled.\n", utils.Styled("→", utils.StyleInfo))
			return nil

		default:
			fmt.Printf("%s Invalid choice. Please enter 'e', 'a', or 'd'.\n", utils.Styled("✗", utils.StyleError))
		}
	}
}

func isGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

func getGitStagedChanges() (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--unified=3")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func getGitUnstagedChanges() (string, error) {
	cmd := exec.Command("git", "diff", "--unified=3")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func getGitStatus() (string, error) {
	cmd := exec.Command("git", "status", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func getGitFileStats(staged bool) (string, error) {
	args := []string{"diff", "--stat"}
	if staged {
		args = append(args, "--cached")
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func getUntrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		return []string{}, nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return files, nil
}

func getUntrackedFilesContent() (string, error) {
	files, err := getUntrackedFiles()
	if err != nil || len(files) == 0 {
		return "", err
	}

	var content strings.Builder
	content.WriteString("Untracked files (new files):\n")

	for _, file := range files {
		if file == "" {
			continue
		}

		content.WriteString(fmt.Sprintf("\n+++ New file: %s\n", file))

		// Read file content
		fileContent, err := os.ReadFile(file)
		if err != nil {
			content.WriteString(fmt.Sprintf("(Unable to read file: %v)\n", err))
			continue
		}

		// Check if file is binary
		if isBinaryContent(fileContent) {
			content.WriteString("(Binary file)\n")
			continue
		}

		// Limit content to first 100 lines for very large files
		lines := strings.Split(string(fileContent), "\n")
		maxLines := 100
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			content.WriteString(strings.Join(lines, "\n"))
			content.WriteString(fmt.Sprintf("\n... (truncated, showing first %d lines)\n", maxLines))
		} else {
			content.WriteString(string(fileContent))
		}
	}

	return content.String(), nil
}

func isBinaryContent(content []byte) bool {
	// Simple heuristic: check for null bytes in first 8KB
	checkSize := 8192
	if len(content) < checkSize {
		checkSize = len(content)
	}

	for i := 0; i < checkSize; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

func stageAllChanges() error {
	cmd := exec.Command("git", "add", "-A")
	return cmd.Run()
}

// Helper: convert untracked content to a pseudo diff
func buildPseudoDiffForUntracked(untrackedFilesContent string) string {
	lines := strings.Split(untrackedFilesContent, "\n")
	var out []string
	currentFile := ""

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r\n")
		trimmedLine := strings.TrimSpace(line)
		if fileName, found := strings.CutPrefix(trimmedLine, "+++ New file: "); found {
			currentFile = strings.TrimSpace(fileName)
			out = append(out, fmt.Sprintf("diff --git a/%s b/%s", currentFile, currentFile))
			out = append(out, "new file mode 100644")
			out = append(out, "index 0000000..e69de29")
			out = append(out, "--- /dev/null")
			out = append(out, fmt.Sprintf("+++ b/%s", currentFile))
			continue
		}
		if currentFile == "" {
			continue
		}
		out = append(out, fmt.Sprintf("+%s", line))
	}

	return strings.Join(out, "\n")
}

// Replace your buildCommitPrompt with this version
func buildCommitPrompt(changes string, isStaged bool, fileStats string, gitStatus string, untrackedFiles string) string {
	changeType := "unstaged changes"
	if isStaged {
		changeType = "staged changes"
	}

	var promptParts []string

	// Build pseudo diff for untracked and place it before main diff
	pseudoUntracked := ""
	if strings.TrimSpace(untrackedFiles) != "" {
		pseudoUntracked = buildPseudoDiffForUntracked(untrackedFiles)
	}

	promptParts = append(promptParts, fmt.Sprintf(
		`You are generating a Conventional Commit message for the following git %s.

Strict output format (return ONLY the commit message, no extra text, no markdown):
- Subject: <type>(<scope>): <imperative subject under 72 chars>
- Optional body: add more detail if non-trivial; separate by a single blank line
- Optional footer: BREAKING CHANGE: <details> and/or references (e.g., Closes #123)

Conventional Commit types:
feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert

Rules:
- Use imperative mood: "add", "fix", "refactor", not "added"/"fixes"/"refactored"
- Prefer a specific <scope> based on the main area changed (package/module/feature); omit scope if unclear
- Keep the subject concise, concrete, and user-facing when applicable
- If multiple files change, choose the dominant scope (by impact), not just file count
- Call out BREAKING CHANGE in footer if it changes public API, behavior, or contracts
- Avoid mentioning internal ref names, usernames, or secrets
- Deprioritize non-functional changes unless significant (e.g., docs content rewrite > typo)
- Avoid noise from lock files, formatting-only changes, regenerated code, or transient diffs
- Do not include code snippets in the body; summarize behavior/intent
- If the changes are trivial (e.g., whitespace-only), use a minimal appropriate type and subject
- If untracked files exist, treat them as new additions and include them in commit intent

Prioritization heuristics:
- Prioritize application and library code over config/docs: .ts/.tsx/.js/.jsx/.go/.py/.rb/.java/.kt/.swift/.rs/.c/.cpp/.cs
- Deprioritize purely non-code files: .md .txt .json .yaml/.yml .toml .lock .svg .png .jpg
- Prioritize changes in untracked files
- Treat test-only changes as type "test" unless they imply a bug fix in code
- For dependency bumps (package.json, go.mod, etc.), use "build" or "chore" depending on impact
- For performance-related changes with measurable improvements, use "perf"

Data provided below may include diff, file stats, git status, and untracked content.
Ignore irrelevant sections, binary data, or malformed snippets. Extract intent and summarize changes.
`, changeType))

	if fileStats != "" {
		promptParts = append(promptParts, fmt.Sprintf("File Statistics:\n%s\n", strings.TrimSpace(fileStats)))
	}
	if gitStatus != "" {
		promptParts = append(promptParts, fmt.Sprintf("Git Status:\n%s\n", strings.TrimSpace(gitStatus)))
	}

	// Put untracked pseudo-diff BEFORE the main diff
	if pseudoUntracked != "" {
		promptParts = append(promptParts, fmt.Sprintf("Diff (Untracked new files):\n%s\n", pseudoUntracked))
	}
	if changes != "" {
		promptParts = append(promptParts, fmt.Sprintf("Diff:\n%s\n", strings.TrimSpace(changes)))
	}

	promptParts = append(promptParts, "Return ONLY the commit message. No explanations, no extra lines before or after, no code blocks.")

	return strings.Join(promptParts, "\n")
}

// isConventionalSubjectLine checks if the first line matches Conventional Commits: <type>(<scope>): <subject> or <type>: <subject>
func isConventionalSubjectLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	types := []string{"feat","fix","docs","style","refactor","test","chore","perf","ci","build","revert"}
	for _, t := range types {
		if strings.HasPrefix(line, t+": ") || strings.HasPrefix(line, t+"(") {
			// If scope form, ensure it ends with "): "
			if strings.HasPrefix(line, t+"(") {
				closeIdx := strings.Index(line, "): ")
				return closeIdx > len(t) // basic check
			}
			return true
		}
	}
	return false
}

// normalizeCommitMessage trims wrappers and ensures at most subject + one body + optional footer.
func normalizeCommitMessage(raw string) string {
	if raw == "" {
		return ""
	}

	// Strip common wrappers and quotes
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "git commit -m ")
	s = strings.Trim(s, "'\"")
	s = strings.TrimSpace(s)

	// Strip code fences if present
	if strings.HasPrefix(s, "```") && strings.HasSuffix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}

	// Remove leading labels
	s = strings.TrimPrefix(s, "Commit message: ")
	s = strings.TrimPrefix(s, "Message: ")
	s = strings.TrimSpace(s)

	// Remove any leading "Explanation:" blocks
	if idx := strings.Index(s, "Explanation:"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}

	// Collapse excessive blank lines
	lines := strings.Split(s, "\n")
	var cleaned []string
	for _, l := range lines {
		cleaned = append(cleaned, strings.TrimRight(l, " "))
	}
	s = strings.Join(cleaned, "\n")
	s = strings.TrimSpace(s)

	// Split into paragraphs: subject, body, footer (optional)
	paras := strings.Split(s, "\n\n")
	if len(paras) == 0 {
		return ""
	}

	subject := strings.TrimSpace(paras[0])
	if !isConventionalSubjectLine(subject) {
		// If the first line is part of a multi-line subject (some models break lines),
		// join up to the first blank line and re-check.
		joinedFirst := strings.ReplaceAll(paras[0], "\n", " ")
		joinedFirst = strings.Join(strings.Fields(joinedFirst), " ")
		if isConventionalSubjectLine(joinedFirst) {
			subject = joinedFirst
		} else {
			// Not a conventional subject; reject
			return ""
		}
	}

	// Enforce 72-char cap on subject (soft trim)
	if len(subject) > 72 {
		subject = subject[:72]
	}

	// Keep a single body block if present and not an explanation/code fence
	var body string
	if len(paras) > 1 {
		b := strings.TrimSpace(paras[1])
		if b != "" && !strings.HasPrefix(b, "Explanation:") && !strings.HasPrefix(b, "```") {
			body = b
		}
	}

	// Optional footer: allow lines containing BREAKING CHANGE or references like "Closes #123"
	var footer string
	if len(paras) > 2 {
		f := strings.TrimSpace(paras[2])
		if f != "" && (strings.HasPrefix(f, "BREAKING CHANGE:") ||
			strings.HasPrefix(f, "Closes ") ||
			strings.HasPrefix(f, "Refs ") ||
			strings.HasPrefix(f, "Relates ")) {
			footer = f
		}
	}

	// Reassemble strictly
	if body != "" && footer != "" {
		return subject + "\n\n" + body + "\n\n" + footer
	}
	if body != "" {
		return subject + "\n\n" + body
	}
	if footer != "" {
		return subject + "\n\n" + footer
	}
	return subject
}

// Extract only from Command; if invalid and verbose is allowed, try Explanation strictly.
func extractCommitMessage(command string, explanation string, allowExplanation bool) string {
	// First, try to normalize Command
	if msg := normalizeCommitMessage(command); msg != "" {
		return msg
	}
	// If Command is unusable and we allow explanation, try explanation
	if allowExplanation {
		if msg := normalizeCommitMessage(explanation); msg != "" {
			return msg
		}
	}
	return ""
}

func enforceSingleCommitMessage(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// Remove leading "Commit message:" or "Message:" labels
	s = strings.TrimPrefix(s, "Commit message: ")
	s = strings.TrimPrefix(s, "Message: ")
	s = strings.TrimSpace(s)

	// Split paragraphs
	parts := strings.Split(s, "\n\n")
	subject := strings.TrimSpace(parts[0])

	// If the first line is not a conventional subject, try to salvage by taking its first sentence as subject
	if !isConventionalSubjectLine(subject) {
		// Try to take the first line only
		firstLine := subject
		if idx := strings.Index(firstLine, ". "); idx > 0 {
			firstLine = firstLine[:idx+1]
		}
		// If it still isn't conventional, discard non-conforming lines
		if !isConventionalSubjectLine(firstLine) {
			// Not recoverable; return empty to trigger fallback (verbose only)
			return ""
		}
		subject = firstLine
	}

	// Subject length cap
	if len(subject) > 72 {
		subject = subject[:72]
	}

	// Optional body: only keep one block, without code fences or "Explanation:"
	var body string
	if len(parts) > 1 {
		b := strings.TrimSpace(parts[1])
		if b != "" && !strings.HasPrefix(b, "```") && !strings.HasPrefix(b, "Explanation:") {
			body = b
		}
	}

	if body != "" {
		return subject + "\n\n" + body
	}
	return subject
}

func openEditorForCommit(initialMessage string) (string, error) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "COMMIT_EDITMSG_*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpfile.Name())

	// Write initial message with helpful comments
	header := "# Edit your commit message above\n"
	header += "# Lines starting with '#' will be ignored\n"
	header += "# An empty message aborts the commit\n"
	if _, err := tmpfile.WriteString(initialMessage + "\n\n" + header); err != nil {
		return "", err
	}
	tmpfile.Close()

	// Get editor from environment or use sensible defaults
	editor := os.Getenv("GIT_EDITOR")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi" // fallback to vi
	}

	// Open editor
	cmd := exec.Command("sh", "-c", fmt.Sprintf("%s %s", editor, tmpfile.Name()))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Read edited content
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}

	// Remove comment lines and trim
	lines := strings.Split(string(content), "\n")
	var finalLines []string
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") && strings.TrimSpace(line) != "" {
			finalLines = append(finalLines, line)
		}
	}

	result := strings.TrimSpace(strings.Join(finalLines, "\n"))

	// Check if message is empty after editing
	if result == "" {
		return "", fmt.Errorf("commit message is empty, aborting")
	}

	return result, nil
}

func applyCommit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startSpinner(message string) func() {
	spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	stopChan := make(chan bool)

	go func() {
		i := 0
		for {
			select {
			case <-stopChan:
				// Clear the spinner line
				fmt.Printf("\r%s\r", strings.Repeat(" ", len(message)+10))
				return
			default:
				fmt.Printf("\r%s %s ",
					utils.Styled(spinnerChars[i%len(spinnerChars)], utils.StyleInfo),
					message)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	return func() {
		stopChan <- true
		time.Sleep(10 * time.Millisecond) // Give time for the goroutine to clear the line
	}
}
