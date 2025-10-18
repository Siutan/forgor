Totally agree — the current “guess-and-scrape” parsing is doing too much work. The simplest way to make this bullet-proof is:

1. **Ask the model for structured output** (strict JSON with `subject`, `body`, `footer`),
2. **Validate & normalise** the JSON,
3. **Assemble** the final message from those fields, with a tiny rules engine.

Below is a drop-in refactor that keeps your existing provider API (`GenerateCommand`) but changes the prompt and parsing path so the code is simpler and far more reliable.

---

# 1) Data model + helpers

````go
// commitmsg.go (new file or near the bottom of commit.go)

package cmd

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

type CommitMessage struct {
	Subject string `json:"subject"`
	Body    string `json:"body,omitempty"`
	Footer  string `json:"footer,omitempty"`
}

// Allow the standard CC types. You can add others if you want.
var conventionalTypes = map[string]struct{}{
	"feat": {}, "fix": {}, "docs": {}, "style": {}, "refactor": {},
	"test": {}, "chore": {}, "perf": {}, "ci": {}, "build": {}, "revert": {},
}

var footerPrefixes = []string{
	"BREAKING CHANGE:", "BREAKING-CHANGE:",
	"Closes ", "Fixes ", "Resolves ", "Refs ", "Relates ",
}

// --- Helpers ---

func trimRunes(s string, max int) string {
	if s == "" || max <= 0 { return "" }
	if utf8.RuneCountInString(s) <= max { return s }
	r := []rune(s)
	return string(r[:max])
}

// <type>(<scope>): <subject>   OR   <type>: <subject>
var reSubject = regexp.MustCompile(`^(?P<type>[a-z]+)(?:\((?P<scope>[^)]+)\))?:\s+(?P<rest>.+)$`)

func isConventionalSubjectLine(line string) bool {
	m := reSubject.FindStringSubmatch(strings.TrimSpace(line))
	if len(m) == 0 { return false }
	t := m[reSubject.SubexpIndex("type")]
	_, ok := conventionalTypes[t]
	return ok
}

// Clean model-ish junk.
func scrub(s string) string {
	if s == "" { return s }
	// Remove accidental code-fences or labels
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(strings.TrimPrefix(s, "Commit message:"))
	s = strings.TrimSpace(strings.TrimPrefix(s, "Message:"))
	return strings.TrimSpace(s)
}

func normaliseFooter(s string) string {
	s = scrub(s)
	if s == "" { return "" }
	// Keep only recognised footer styles (first paragraph).
	parts := strings.SplitN(s, "\n\n", 2)
	f := strings.TrimSpace(parts[0])
	for _, p := range footerPrefixes {
		if strings.HasPrefix(f, p) {
			return f
		}
	}
	return ""
}

func normaliseBody(s string) string {
	s = scrub(s)
	if s == "" { return "" }
	// Strip leading accidental labels/fences inside body too.
	return s
}

func validateAndFix(cm *CommitMessage) error {
	cm.Subject = scrub(cm.Subject)
	cm.Body = normaliseBody(cm.Body)
	cm.Footer = normaliseFooter(cm.Footer)

	// If subject got split across lines, join it.
	if !isConventionalSubjectLine(cm.Subject) && strings.Contains(cm.Subject, "\n") {
		joined := strings.Join(strings.Fields(strings.ReplaceAll(cm.Subject, "\n", " ")), " ")
		if isConventionalSubjectLine(joined) {
			cm.Subject = joined
		}
	}

	// Enforce conventional subject and 72-rune cap.
	if !isConventionalSubjectLine(cm.Subject) {
		return errors.New("subject is not a valid Conventional Commit line")
	}
	cm.Subject = trimRunes(cm.Subject, 72)

	return nil
}

func formatCommit(cm CommitMessage) string {
	if cm.Footer != "" && cm.Body != "" {
		return cm.Subject + "\n\n" + cm.Body + "\n\n" + cm.Footer
	}
	if cm.Body != "" {
		return cm.Subject + "\n\n" + cm.Body
	}
	if cm.Footer != "" {
		return cm.Subject + "\n\n" + cm.Footer
	}
	return cm.Subject
}
````

---

# 2) Ask the model for JSON (no more heuristic scraping)

```go
// replace buildCommitPrompt(...) with a JSON-spec version
func buildCommitPrompt(changes string, isStaged bool, fileStats string, gitStatus string, untrackedFiles string) string {
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
  "footer": "string, OPTIONAL, only if needed. Allowed prefixes: 'BREAKING CHANGE:', 'BREAKING-CHANGE:', 'Closes ', 'Fixes ', 'Resolves ', 'Refs ', 'Relates '"
}

Conventional Commit types allowed:
feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert

Rules:
- Use imperative mood in subject.
- Prefer a specific <scope> (module/package/feature) if clear; otherwise omit scope.
- Body: summarise intent and impact. No code snippets.
- Footer: include BREAKING* or issue refs only when appropriate.
- If untracked files exist, treat them as new additions and include in intent.
- Ignore binary content and noise (lockfiles, large generated assets).

Choose at most one type and at most one scope.
`)

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

	b.WriteString("\nReturn JSON only.")
	return b.String()
}
```

---

# 3) Robust JSON parsing (first try Command, then Explanation)

````go
// new function to parse the provider output as JSON
func parseCommitJSON(command, explanation string) (*CommitMessage, error) {
	var firstErr error

	try := func(payload string) (*CommitMessage, error) {
		payload = strings.TrimSpace(payload)
		// defensive: strip code fences if the model ignored instructions
		payload = strings.TrimPrefix(payload, "```json")
		payload = strings.TrimPrefix(payload, "```")
		payload = strings.TrimSuffix(payload, "```")
		payload = strings.TrimSpace(payload)

		var cm CommitMessage
		if err := json.Unmarshal([]byte(payload), &cm); err != nil {
			return nil, err
		}
		if err := validateAndFix(&cm); err != nil {
			return nil, err
		}
		return &cm, nil
	}

	if cm, err := try(command); err == nil {
		return cm, nil
	} else {
		firstErr = err
	}

	// Fallback to explanation if allowed by verbose mode or if command failed hard
	if cm, err := try(explanation); err == nil {
		return cm, nil
	}

	return nil, firstErr
}
````

---

# 4) Wire it into your flow (smaller `runCommitGeneration`)

Key changes:

* Use the new JSON prompt.
* Replace `extractCommitMessage`/`normalizeCommitMessage` with `parseCommitJSON` + `formatCommit`.
* (Optional but recommended) switch to `git commit -F tmpfile` as noted.

```go
// inside runCommitGeneration(), after you compute `prompt`:

ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()

response, err := provider.GenerateCommand(ctx, &llm.Request{
	Query:   prompt,
	Context: requestContext,
	Options: llm.RequestOptions{
		IncludeExplanation: commitVerbose,
		MaxTokens:          300,
		Temperature:        0.2, // slightly lower for schema compliance
	},
})
stopSpinner()
if err != nil {
	return fmt.Errorf("failed to generate commit message: %w", err)
}

if commitVerbose {
	fmt.Printf("%s Model raw payload received.\n", utils.Styled("•", utils.StyleSubtle))
}

cm, err := parseCommitJSON(response.Command, response.Explanation)
if err != nil {
	return fmt.Errorf("failed to parse model JSON: %w", err)
}

commitMessage := formatCommit(*cm)
if strings.TrimSpace(commitMessage) == "" {
	return fmt.Errorf("empty commit message after parsing")
}

// display
fmt.Printf("\n%s\n", utils.Divider("GENERATED COMMIT MESSAGE", utils.StyleSuccess))
fmt.Printf("%s\n", utils.Styled(commitMessage, utils.StyleCommand))
fmt.Printf("%s\n\n", utils.Divider("", utils.StyleSuccess))

if commitVerbose && response.Explanation != "" {
	fmt.Printf("%s %s\n\n", utils.Styled("Explanation:", utils.StyleInfo), response.Explanation)
}
```

*(If you keep your existing interactive apply/edit loop, no other changes are needed.)*

---

# 5) Why this fixes your “subject/body/footer” weirdness

* We **don’t** try to infer structure from free-form text any more.
* The model is forced to choose a **single subject string** that already conforms to the Conventional Commit shape.
* We do **one** validation step (`validateAndFix`) with a tiny, explicit set of rules.
* The footer is constrained to known prefixes (so you won’t get the footer accidentally glued to the subject like `footer: %srefactor(...)...` again).
* The final assembly is 3 lines of code with zero ambiguity.

---

# 6) (Nice to have) Table-driven tests

If you’re up for it, a tiny test suite will lock this in:

```go
// commitmsg_test.go
package cmd

import "testing"

func TestValidateAndFix(t *testing.T) {
	tests := []struct{
		name string
		in   CommitMessage
		ok   bool
	}{
		{"ok_no_scope", CommitMessage{Subject:"refactor: simplify parsing"}, true},
		{"ok_scope", CommitMessage{Subject:"feat(api): add v2 endpoint"}, true},
		{"bad_type", CommitMessage{Subject:"update: something"}, false},
		{"broken_lines", CommitMessage{Subject:"refactor(api):\n simplify parser"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T){
			cm := tt.in
			err := validateAndFix(&cm)
			if tt.ok && err != nil { t.Fatalf("unexpected err: %v", err) }
			if !tt.ok && err == nil { t.Fatalf("expected error") }
		})
	}
}

func TestFormatCommit(t *testing.T) {
	cm := CommitMessage{
		Subject: "fix(router): handle 404",
		Body:    "Avoid nil deref when route is missing.",
		Footer:  "Closes #123",
	}
	got := formatCommit(cm)
	want := "fix(router): handle 404\n\nAvoid nil deref when route is missing.\n\nCloses #123"
	if got != want { t.Fatalf("got:\n%s\nwant:\n%s", got, want) }
}
```

---

## Summary

* **Switch to JSON** from the model → **tiny validation** → **straight assembly**.
* No more brittle regex slicing, no more “explanation fallback” gymnastics, and it resolves the “footer glued onto subject” confusion you’re seeing.

If you want, I can also fold in the `git commit -F` change and secret-safe truncation we discussed, but the above gets your parsing/refactor over the line first.
