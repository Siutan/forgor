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
	"golang.org/x/term"
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
	
	branchName, err := getBranchName()
	if err != nil {
		return fmt.Errorf("failed to get branch name: %w", err)
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

	prompt := buildCommitPrompt(branchName, changes, useStaged, fileStats, gitStatus, untrackedFiles)

	// Generate commit message
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	requestContext := llm.BuildContextFromSystem()

	response, err := provider.GenerateCommand(ctx, &llm.Request{
		Query:   prompt,
		Context: requestContext,
		Options: llm.RequestOptions{
			IncludeExplanation: commitVerbose,
			MaxTokens:          1000,
			Temperature:        0.2,
		},
	})

	stopSpinner()

	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	if commitVerbose {
		// Show a short snippet of the provider payload (single-line, truncated)
		snippet := response.Command
		if strings.TrimSpace(snippet) == "" {
			snippet = response.Explanation
		}
		snippet = strings.TrimSpace(snippet)
		if len([]rune(snippet)) > 200 {
			r := []rune(snippet)
			snippet = string(r[:200]) + "…"
		}
		oneLine := strings.ReplaceAll(snippet, "\n", " ")
		fmt.Printf("%s Model payload snippet: %s\n", utils.Styled("•", utils.StyleSubtle), utils.Styled(oneLine, utils.StyleSubtle))
	}

	cm, err := parseCommitJSON(response.Command, response.Explanation)
	var commitMessage string
	if err != nil {
		// Fallback to legacy free-form parsing for robustness
		commitMessage = extractCommitMessage(response.Command, response.Explanation, commitVerbose)
		if strings.TrimSpace(commitMessage) == "" {
			snippet := response.Command
			if strings.TrimSpace(snippet) == "" {
				snippet = response.Explanation
			}
			snippet = strings.TrimSpace(snippet)
			if len([]rune(snippet)) > 200 {
				r := []rune(snippet)
				snippet = string(r[:200]) + "…"
			}
			oneLine := strings.ReplaceAll(snippet, "\n", " ")
			return fmt.Errorf("failed to parse model JSON and no valid free-form commit message was found. payload: %q. err: %w", oneLine, err)
		}
	} else {
		commitMessage = formatCommit(*cm)
		if strings.TrimSpace(commitMessage) == "" {
			return fmt.Errorf("empty commit message after parsing")
		}
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

		// Read a single key without requiring Enter (raw mode), with graceful fallback
		var choice string
		if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
			b := make([]byte, 1)
			_, rerr := os.Stdin.Read(b)
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
			if rerr != nil {
				return fmt.Errorf("failed to read input: %w", rerr)
			}
			choice = strings.ToLower(string(b[0]))
			// Echo the key and newline for UX in raw mode
			fmt.Printf("%s\n", choice)
		} else {
			reader := bufio.NewReader(os.Stdin)
			input, rerr := reader.ReadString('\n')
			if rerr != nil {
				return fmt.Errorf("failed to read input: %w", rerr)
			}
			choice = strings.ToLower(strings.TrimSpace(input))
		}

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

				// Single-key read for y/n confirmation
				stageChoice := ""
				if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
					b := make([]byte, 1)
					n, rerr := os.Stdin.Read(b)
					_ = term.Restore(int(os.Stdin.Fd()), oldState)
					if rerr != nil {
						return fmt.Errorf("failed to read input: %w", rerr)
					}
					if n == 0 {
						return fmt.Errorf("failed to read input: no data received")
					}
					stageChoice = strings.ToLower(string(b[0]))
					fmt.Printf("%s\n", stageChoice)
				} else {
					r := bufio.NewReader(os.Stdin)
					s, rerr := r.ReadString('\n')
					if rerr != nil {
						return fmt.Errorf("failed to read input: %w", rerr)
					}
					stageChoice = strings.ToLower(strings.TrimSpace(s))
				}

				if stageChoice == "y" {
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

func getBranchName() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
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
func buildCommitPrompt(branchName string, changes string, isStaged bool, fileStats string, gitStatus string, untrackedFiles string) string {
	changeType := "unstaged changes"
	if isStaged {
		changeType = "staged changes"
	}

	var b strings.Builder
	b.WriteString(
`You are generating a Conventional Commit message for the following git ` + changeType + `.

Return JSON ONLY, no markdown, no prose. Schema:

{
  "subject": "string, REQUIRED. Must be '<type>(<scope>): <imperative subject>' or '<type>: <imperative subject>' and <= 72 chars",
  "body": "string, OPTIONAL, a single paragraph with details (no code blocks)",
  "footer": "string, OPTIONAL, only if needed. Allowed prefixes: 'BREAKING CHANGE:', 'BREAKING-CHANGE:'"
}

Conventional Commit types allowed:
feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert

Rules:
- Use imperative mood in subject.
- Prefer a specific <scope> (module/package/feature) if clear; otherwise omit scope.
- Body: summarise intent and impact. No code snippets. Use short dot points in order of importance.
- Footer: include issue reference if applicable, usually in branch name.
- If untracked files exist, treat them as new additions and include in intent.
- Ignore binary content and noise (lockfiles, large generated assets).

Choose at most one type and at most one scope.
`)

	b.WriteString("\nBranch Name:\n")
	b.WriteString(branchName)
	b.WriteString("\n")

	if fileStats != "" {
		b.WriteString("\nFile Statistics:\n")
		b.WriteString(strings.TrimSpace(fileStats))
		b.WriteString("\n")
	}
	if gitStatus != "" {
		b.WriteString("\nGit Status:\n")
		b.WriteString(strings.TrimSpace(gitStatus))
		b.WriteString("\n")
	}

	// Put untracked pseudo-diff before main diff
	if s := strings.TrimSpace(untrackedFiles); s != "" {
		pseudo := buildPseudoDiffForUntracked(s)
		if pseudo != "" {
			b.WriteString("\nDiff (Untracked new files):\n")
			b.WriteString(pseudo)
			b.WriteString("\n")
		}
	}
	if changes != "" {
		b.WriteString("\nDiff:\n")
		b.WriteString(strings.TrimSpace(changes))
		b.WriteString("\n")
	}

	b.WriteString("\nReturn JSON only. Begin with '{' and end with '}'. No code fences, no extra text.")
	return b.String()
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
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	// Strip common wrappers and quotes
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "git commit -m ")
	s = strings.Trim(s, "'\"")
	s = strings.TrimSpace(s)

	// Strip code fences if present (handle opening and closing fences)
	if strings.HasPrefix(s, "```") {
		if last := strings.LastIndex(s, "```"); last > 3 {
			s = strings.TrimSpace(s[3:last])
		}
	}

	// Remove leading labels (case-sensitive forms used by models)
	s = strings.TrimPrefix(s, "Commit message: ")
	s = strings.TrimPrefix(s, "Message: ")
	s = strings.TrimSpace(s)

	// Remove any trailing "Explanation:" blocks (and anything after)
	if idx := strings.Index(s, "Explanation:"); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}

	// Collapse excessive blank lines and trim trailing spaces on each line
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}

	// Collapse consecutive blank lines to a single blank line
	var collapsed []string
	blank := false
	for i := range lines {
		if strings.TrimSpace(lines[i]) == "" {
			if !blank {
				collapsed = append(collapsed, "")
				blank = true
			}
		} else {
			collapsed = append(collapsed, lines[i])
			blank = false
		}
	}
	s = strings.TrimSpace(strings.Join(collapsed, "\n"))

	// Split into paragraphs: subject, body, footer (optional)
	paras := strings.Split(s, "\n\n")
	if len(paras) == 0 || strings.TrimSpace(paras[0]) == "" {
		return ""
	}

	subject := strings.TrimSpace(paras[0])
	if !isConventionalSubjectLine(subject) {
		// Try to join broken lines in the first paragraph into a single line and re-check
		joined := strings.ReplaceAll(paras[0], "\n", " ")
		joined = strings.Join(strings.Fields(joined), " ")
		if isConventionalSubjectLine(joined) {
			subject = joined
		} else {
			// As a last attempt, take only the very first non-empty line
			firstLine := subject
			if idx := strings.Index(firstLine, "\n"); idx >= 0 {
				firstLine = firstLine[:idx]
			}
			firstLine = strings.TrimSpace(firstLine)
			if isConventionalSubjectLine(firstLine) {
				subject = firstLine
			} else {
				// Not a conventional subject; reject
				return ""
			}
		}
	}

	// Enforce 72-char cap on subject using runes (soft trim)
	rs := []rune(subject)
	if len(rs) > 72 {
		subject = string(rs[:72])
	}

	// Keep a single body block if present and not an explanation/code fence
	var body string
	if len(paras) > 1 {
		b := strings.TrimSpace(paras[1])
		if b != "" && !strings.HasPrefix(b, "Explanation:") && !strings.HasPrefix(b, "```") {
			body = b
		}
	}

	var footer string
	if len(paras) > 2 {
		f := strings.TrimSpace(strings.Join(paras[2:], "\n\n"))
		if f != "" {
			fLines := strings.Split(f, "\n")
			var footerLines []string
			for i := range fLines {
				l := strings.TrimSpace(fLines[i])
				if l == "" {
					continue
				}
				low := strings.ToLower(l)
				if strings.HasPrefix(low, "breaking change:") || strings.HasPrefix(low, "breaking-change:") {
					footerLines = append(footerLines, l)
					continue
				}
				if strings.HasPrefix(low, "closes") || strings.HasPrefix(low, "closes:") {
					footerLines = append(footerLines, l)
					continue
				}
				if strings.Contains(low, "closes #") {
					footerLines = append(footerLines, l)
					continue
				}
			}
			if len(footerLines) > 0 {
				footer = strings.Join(footerLines, "\n")
			}
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
	tmpfile, err := os.CreateTemp("", "COMMIT_MSG_*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(message); err != nil {
		tmpfile.Close()
		return err
	}
	tmpfile.Close()

	cmd := exec.Command("git", "commit", "-F", tmpfile.Name())
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
