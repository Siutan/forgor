package daemon

import (
	"regexp"
	"strings"
	"sync"
)

type Sanitizer struct {
	patterns []Pattern
	mu       sync.RWMutex
}

type Pattern struct {
	Name        string
	Regex       *regexp.Regexp
	Replacement string
	Severity    string
}

type Detection struct {
	Pattern  string
	Count    int
	Severity string
}

func defaultPatterns() []Pattern {
	return []Pattern{
		{Name: "AWS Access Key", Regex: regexp.MustCompile(`AKIA[0-9A-Z]{16}`), Replacement: "[AWS_ACCESS_KEY_REDACTED]", Severity: "high"},
		{Name: "JWT Token", Regex: regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`), Replacement: "[JWT_TOKEN_REDACTED]", Severity: "high"},
		{Name: "Password", Regex: regexp.MustCompile(`(?i)(password|passwd|pwd)[\s:=]+['\"]?([^\s'\"]+)['\"]?`), Replacement: "[PASSWORD_REDACTED]", Severity: "high"},
		{Name: "GitHub Token", Regex: regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`), Replacement: "[GITHUB_TOKEN_REDACTED]", Severity: "high"},
		{Name: "Email", Regex: regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`), Replacement: "[EMAIL_REDACTED]", Severity: "low"},
		{Name: "IPv4", Regex: regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`), Replacement: "[IP_REDACTED]", Severity: "low"},
	}
}

func NewSanitizer() *Sanitizer {
	s := &Sanitizer{patterns: defaultPatterns()}
	return s
}

func (s *Sanitizer) Sanitize(text string) (string, []Detection) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := text
	var dets []Detection
	for _, p := range s.patterns {
		matches := p.Regex.FindAllStringIndex(out, -1)
		if len(matches) == 0 {
			continue
		}
		dets = append(dets, Detection{Pattern: p.Name, Count: len(matches), Severity: p.Severity})
		out = p.Regex.ReplaceAllString(out, p.Replacement)
	}
	return out, dets
}

func (s *Sanitizer) DetectSensitiveCommand(cmd string) bool {
	lowered := strings.ToLower(cmd)
	sensitive := []string{"vault login", "aws configure", "docker login", "gcloud auth"}
	for _, v := range sensitive {
		if strings.Contains(lowered, v) {
			return true
		}
	}
	return false
}
