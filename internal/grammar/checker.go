package grammar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const defaultLanguageToolURL = "https://api.languagetool.org/v2/check"

var (
	fencedCode   = regexp.MustCompile("(?s)```.*?```")
	inlineCode   = regexp.MustCompile("`[^`]+`")
	images       = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	links        = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`)
	headings     = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	blockquote   = regexp.MustCompile(`(?m)^>\s?`)
	listMarkers  = regexp.MustCompile(`(?m)^[\*\-\+]\s+`)
	orderedList  = regexp.MustCompile(`(?m)^\d+\.\s+`)
	hr           = regexp.MustCompile(`(?m)^(\*{3,}|-{3,}|_{3,})\s*$`)
	emphasis     = regexp.MustCompile(`[*_]{1,3}([^*_]+)[*_]{1,3}`)
)

type Checker struct {
	client  *http.Client
	baseURL string
}

func NewChecker(baseURL string) *Checker {
	if baseURL == "" {
		baseURL = defaultLanguageToolURL
	}
	return &Checker{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: baseURL,
	}
}

type Match struct {
	Message     string   `json:"message"`
	ShortMessage string  `json:"short_message,omitempty"`
	Offset      int      `json:"offset"`
	Length      int      `json:"length"`
	Context     string   `json:"context"`
	Suggestions []string `json:"suggestions"`
	RuleID      string   `json:"rule_id,omitempty"`
}

type CheckResult struct {
	Text    string  `json:"text"`
	Matches []Match `json:"matches"`
}

type ltResponse struct {
	Matches []struct {
		Message      string `json:"message"`
		ShortMessage string `json:"shortMessage"`
		Offset       int    `json:"offset"`
		Length       int    `json:"length"`
		Context      struct {
			Text string `json:"text"`
		} `json:"context"`
		Replacements []struct {
			Value string `json:"value"`
		} `json:"replacements"`
		Rule struct {
			ID string `json:"id"`
		} `json:"rule"`
	} `json:"matches"`
}

func StripMarkdown(md string) string {
	s := md
	s = fencedCode.ReplaceAllString(s, " ")
	s = inlineCode.ReplaceAllString(s, " ")
	s = images.ReplaceAllString(s, " ")
	s = links.ReplaceAllString(s, " ")
	s = headings.ReplaceAllString(s, "")
	s = blockquote.ReplaceAllString(s, "")
	s = listMarkers.ReplaceAllString(s, "")
	s = orderedList.ReplaceAllString(s, "")
	s = hr.ReplaceAllString(s, "")
	s = emphasis.ReplaceAllString(s, "$1")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func (c *Checker) Check(ctx context.Context, text, language string) (CheckResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return CheckResult{Text: "", Matches: []Match{}}, nil
	}
	if language == "" {
		language = "en-US"
	}

	form := url.Values{}
	form.Set("text", text)
	form.Set("language", language)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, strings.NewReader(form.Encode()))
	if err != nil {
		return CheckResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return CheckResult{}, fmt.Errorf("grammar service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CheckResult{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return CheckResult{}, fmt.Errorf("grammar service returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var lt ltResponse
	if err := json.Unmarshal(body, &lt); err != nil {
		return CheckResult{}, fmt.Errorf("parse grammar response: %w", err)
	}

	result := CheckResult{Text: text, Matches: make([]Match, 0, len(lt.Matches))}
	for _, m := range lt.Matches {
		suggestions := make([]string, 0, len(m.Replacements))
		for _, r := range m.Replacements {
			if r.Value != "" {
				suggestions = append(suggestions, r.Value)
			}
			if len(suggestions) >= 5 {
				break
			}
		}
		result.Matches = append(result.Matches, Match{
			Message:      m.Message,
			ShortMessage: m.ShortMessage,
			Offset:       m.Offset,
			Length:       m.Length,
			Context:      strings.TrimSpace(m.Context.Text),
			Suggestions:  suggestions,
			RuleID:       m.Rule.ID,
		})
	}
	if result.Matches == nil {
		result.Matches = []Match{}
	}
	return result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
