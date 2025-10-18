package cmd

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

// CommitMessage is a structured representation of a Conventional Commit.
type CommitMessage struct {
	Subject string `json:"subject"`
	Body    string `json:"body,omitempty"`
	Footer  string `json:"footer,omitempty"`
}

// Allowed Conventional Commit types.
// You can extend this if your workflow needs custom types.
var conventionalTypes = map[string]struct{}{
	"feat": {}, "fix": {}, "docs": {}, "style": {}, "refactor": {},
	"test": {}, "chore": {}, "perf": {}, "ci": {}, "build": {}, "revert": {},
}

// Recognized footer prefixes. Only these are allowed in the "footer".
var footerPrefixes = []string{
	"BREAKING CHANGE:", "BREAKING-CHANGE:",
}

// trimRunes trims a string to a maximum number of runes.
func trimRunes(s string, max int) string {
	if s == "" || max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max])
}

// <type>(<scope>): <subject>   OR   <type>: <subject>
var reSubject = regexp.MustCompile(`^(?P<type>[a-z]+)(?:\((?P<scope>[^)]+)\))?:\s+(?P<rest>.+)$`)

// isConventionalSubjectJSONLine checks for a valid Conventional Commit subject.
// Note: Named to avoid colliding with existing helpers in commit.go.
func isConventionalSubjectJSONLine(line string) bool {
	m := reSubject.FindStringSubmatch(strings.TrimSpace(line))
	if len(m) == 0 {
		return false
	}
	t := m[reSubject.SubexpIndex("type")]
	_, ok := conventionalTypes[t]
	return ok
}

// scrub removes common LLM artifacts (code fences, labels, etc.)
func scrub(s string) string {
	if s == "" {
		return s
	}
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(strings.TrimPrefix(s, "Commit message:"))
	s = strings.TrimSpace(strings.TrimPrefix(s, "Message:"))
	return strings.TrimSpace(s)
}

// normaliseFooter ensures only recognized footer styles and returns the first
// paragraph if valid; otherwise returns an empty string.
func normaliseFooter(s string) string {
	s = scrub(s)
	if s == "" {
		return ""
	}
	parts := strings.SplitN(s, "\n\n", 2)
	f := strings.TrimSpace(parts[0])
	for _, p := range footerPrefixes {
		if strings.HasPrefix(f, p) {
			return f
		}
	}
	return ""
}

// normaliseBody scrubs and returns the body without extra transformations.
func normaliseBody(s string) string {
	s = scrub(s)
	if s == "" {
		return ""
	}
	return s
}

// validateAndFix validates and lightly normalizes a CommitMessage.
// - scrubs fields
// - optionally joins multi-line subject if it appears split
// - enforces conventional subject line and 72-rune cap
func validateAndFix(cm *CommitMessage) error {
	cm.Subject = scrub(cm.Subject)
	cm.Body = normaliseBody(cm.Body)
	cm.Footer = normaliseFooter(cm.Footer)

	// If subject got split across lines, join it.
	if !isConventionalSubjectJSONLine(cm.Subject) && strings.Contains(cm.Subject, "\n") {
		joined := strings.Join(strings.Fields(strings.ReplaceAll(cm.Subject, "\n", " ")), " ")
		if isConventionalSubjectJSONLine(joined) {
			cm.Subject = joined
		}
	}

	// Enforce conventional subject and 72-rune cap.
	if !isConventionalSubjectJSONLine(cm.Subject) {
		return errors.New("subject is not a valid Conventional Commit line")
	}
	cm.Subject = trimRunes(cm.Subject, 72)

	return nil
}

// formatCommit assembles a final commit message from structured fields.
// It renders as subject, optionally body, optionally footer, separated by blank lines.
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

// parseCommitJSON attempts to parse LLM output as JSON CommitMessage.
// It tries the command payload first, then the explanation payload.
// It also defensively strips code fences if present.
// Additionally, it can extract a JSON object from mixed payloads using brace matching.
func extractJSONObject(payload string) (string, bool) {
	s := strings.TrimSpace(payload)
	// Strip common code fences if present
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	start := strings.Index(s, "{")
	if start == -1 {
		return "", false
	}

	inString := false
	escape := false
	depth := 0
	for i, r := range s[start:] {
		if inString {
			switch {
			case escape:
				// Previous character was an escape; current char is consumed literally
				escape = false
			case r == '\\':
				// Enter escape mode so the next character is treated literally
				escape = true
			case r == '"':
				// Closing quote ends the string
				inString = false
			}
			continue
		}

		// Normal parse outside strings
		switch r {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end := start + i + 1
				return s[start:end], true
			}
		}
	}

	return "", false
}

// extractQuotedValueLoose finds a quoted value for a key like subject/body/footer in JSON-like text.
func extractQuotedValueLoose(s, key string) string {
	k := regexp.QuoteMeta(key)
	reD := regexp.MustCompile(`(?is)["']?` + k + `["']?\s*:\s*"(.*?)"`)
	if m := reD.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	reS := regexp.MustCompile(`(?is)["']?` + k + `["']?\s*:\s*'(.*?)'`)
	if m := reS.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// parseLooseQuotedFields extracts subject/body/footer by looking for quoted fields in JSON-like text.
func parseLooseQuotedFields(payload string) (*CommitMessage, error) {
	s := strings.TrimSpace(payload)
	subject := extractQuotedValueLoose(s, "subject")
	if subject == "" {
		return nil, errors.New("subject not found in loose payload")
	}
	body := extractQuotedValueLoose(s, "body")
	footer := extractQuotedValueLoose(s, "footer")
	cm := CommitMessage{
		Subject: subject,
		Body:    body,
		Footer:  footer,
	}
	if err := validateAndFix(&cm); err != nil {
		return nil, err
	}
	return &cm, nil
}

// parseConventionalFromText scans for a Conventional Commit subject in free-form text and builds message parts.
func parseConventionalFromText(payload string) (*CommitMessage, error) {
	s := scrub(payload)
	if s == "" {
		return nil, errors.New("empty payload")
	}
	lines := strings.Split(s, "\n")
	subjIdx := -1
	for i, ln := range lines {
		l := strings.TrimSpace(ln)
		if isConventionalSubjectJSONLine(l) {
			subjIdx = i
			break
		}
	}
	if subjIdx == -1 {
		return nil, errors.New("no conventional subject found")
	}
	subject := strings.TrimSpace(lines[subjIdx])
	rest := strings.TrimSpace(strings.Join(lines[subjIdx+1:], "\n"))

	body := ""
	footer := ""
	if rest != "" {
		paras := strings.Split(rest, "\n\n")
		if len(paras) > 0 {
			b := strings.TrimSpace(paras[0])
			if b != "" {
				body = b
			}
		}
		if len(paras) > 1 {
			f := normaliseFooter(paras[1])
			if f != "" {
				footer = f
			}
		}
	}

	cm := CommitMessage{
		Subject: subject,
		Body:    body,
		Footer:  footer,
	}
	if err := validateAndFix(&cm); err != nil {
		return nil, err
	}
	return &cm, nil
}

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
		// First, try direct JSON unmarshal
		if err := json.Unmarshal([]byte(payload), &cm); err == nil {
			if err := validateAndFix(&cm); err != nil {
				return nil, err
			}
			return &cm, nil
		} else {
			// If that fails, try to extract a JSON object from within the payload
			if obj, ok := extractJSONObject(payload); ok {
				if err2 := json.Unmarshal([]byte(obj), &cm); err2 == nil {
					if err := validateAndFix(&cm); err != nil {
						return nil, err
					}
					return &cm, nil
				} else {
					// return the original error to allow outer fallbacks
					return nil, err
				}
			}
			// Try a loose parser that extracts quoted fields from JSON-like payloads.
			if cm2, err2 := parseLooseQuotedFields(payload); err2 == nil {
				return cm2, nil
			}
			// Fallback: parse conventional subject/body/footer from free-form text.
			if cm3, err3 := parseConventionalFromText(payload); err3 == nil {
				return cm3, nil
			}
			return nil, err
		}
	}

	if cm, err := try(command); err == nil {
		return cm, nil
	} else {
		firstErr = err
	}

	// Fallback to explanation
	if cm, err := try(explanation); err == nil {
		return cm, nil
	}

	// Last-chance: explicit extraction attempts from both payloads
	if obj, ok := extractJSONObject(command); ok {
		var cm CommitMessage
		if err := json.Unmarshal([]byte(obj), &cm); err == nil {
			if err := validateAndFix(&cm); err == nil {
				return &cm, nil
			}
		}
	}
	if obj, ok := extractJSONObject(explanation); ok {
		var cm CommitMessage
		if err := json.Unmarshal([]byte(obj), &cm); err == nil {
			if err := validateAndFix(&cm); err == nil {
				return &cm, nil
			}
		}
	}

	return nil, firstErr
}
