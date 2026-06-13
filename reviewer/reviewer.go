// Package reviewer runs a structured "pre-human" code review using Claude.
//
// The model is asked to return its review through a *forced tool call*
// (submit_review) rather than free-form prose, so the result is always valid,
// renderable JSON. The same prompt powers Challenge #3 (three improvements plus
// one positive note) and the richer Challenge #1 app output.
package reviewer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultModel is the fast/quality sweet spot. Override with REVIEWER_MODEL,
// e.g. REVIEWER_MODEL=claude-opus-4-8 for the deepest analysis.
const DefaultModel = "claude-sonnet-4-6"

const apiURL = "https://api.anthropic.com/v1/messages"

const systemPrompt = `You are a senior staff engineer doing a pre-human code review. Your job is to catch the things a busy teammate would flag before they spend their time on it, focusing on readability, structure, and maintainability.

Principles:
- Be specific and actionable. Point at the exact construct and say what to do instead, not just that something is "bad".
- Prioritise. Surface the THREE highest-impact improvements first. A reviewer's attention is scarce; do not bury the lede under nitpicks.
- Be proportionate. Match severity to real impact. Do not invent problems to hit a quota, and do not soften a genuine bug.
- Be encouraging and concrete in the positive note — name a real strength in this code, not generic praise.
- Respect the language's idioms and the apparent intent of the snippet. If context is missing, note the assumption instead of guessing wildly.

Always return your review by calling the submit_review tool. Never answer in prose.`

// reviewToolSchema is the JSON Schema for the forced submit_review tool call.
// Keeping it as a raw literal mirrors the wire format exactly.
const reviewToolSchema = `{
  "type": "object",
  "properties": {
    "language": {"type": "string", "description": "Detected programming language."},
    "summary": {"type": "string", "description": "One or two sentences on what the code does and its overall health."},
    "overall_score": {"type": "integer", "minimum": 1, "maximum": 10, "description": "Holistic quality score, 1 (rough) to 10 (ship it)."},
    "positive_note": {"type": "string", "description": "One genuine, specific strength of this code."},
    "improvements": {
      "type": "array", "minItems": 3, "maxItems": 3,
      "description": "The three highest-impact improvements, ordered most important first.",
      "items": {
        "type": "object",
        "properties": {
          "title": {"type": "string"},
          "category": {"type": "string", "enum": ["readability","structure","maintainability","correctness","performance","security"]},
          "severity": {"type": "string", "enum": ["high","medium","low"]},
          "location": {"type": "string", "description": "Line number(s) or the symbol/snippet the finding refers to."},
          "explanation": {"type": "string", "description": "Why it matters, in plain language."},
          "suggestion": {"type": "string", "description": "Concrete fix, with a short code example when useful."}
        },
        "required": ["title","category","severity","explanation","suggestion"]
      }
    },
    "extra_observations": {"type": "array", "items": {"type": "string"}, "description": "Optional minor nits beyond the top three."}
  },
  "required": ["language","summary","overall_score","positive_note","improvements"]
}`

// Improvement is a single ranked finding.
type Improvement struct {
	Title       string `json:"title"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Location    string `json:"location,omitempty"`
	Explanation string `json:"explanation"`
	Suggestion  string `json:"suggestion"`
}

// Result is the structured review returned by the model.
type Result struct {
	Language          string        `json:"language"`
	Summary           string        `json:"summary"`
	OverallScore      int           `json:"overall_score"`
	PositiveNote      string        `json:"positive_note"`
	Improvements      []Improvement `json:"improvements"`
	ExtraObservations []string      `json:"extra_observations,omitempty"`
	Model             string        `json:"-"`
}

// Options tweaks a review. The zero value is valid: the API key falls back to
// ANTHROPIC_API_KEY, the model to REVIEWER_MODEL (or DefaultModel), and an
// http.Client with a sane timeout is created on demand.
type Options struct {
	Language string // optional language hint; the model auto-detects otherwise
	Context  string // optional author context (e.g. "this is a hot loop")
	Model    string
	APIKey   string
	Client   *http.Client
}

// Model resolves the model id from options, then env, then the default.
func (o Options) model() string {
	if o.Model != "" {
		return o.Model
	}
	if m := os.Getenv("REVIEWER_MODEL"); m != "" {
		return m
	}
	return DefaultModel
}

func (o Options) apiKey() string {
	if o.APIKey != "" {
		return o.APIKey
	}
	return os.Getenv("ANTHROPIC_API_KEY")
}

func (o Options) client() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// wire types for the Messages API.
type apiTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type apiRequest struct {
	Model      string            `json:"model"`
	MaxTokens  int               `json:"max_tokens"`
	System     string            `json:"system"`
	Tools      []apiTool         `json:"tools"`
	ToolChoice map[string]string `json:"tool_choice"`
	Messages   []apiMessage      `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []struct {
		Type  string          `json:"type"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Review sends a snippet to Claude and returns a structured review.
func Review(ctx context.Context, code string, opts Options) (*Result, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("no code provided to review")
	}
	key := opts.apiKey()
	if key == "" {
		return nil, fmt.Errorf("no API key: set ANTHROPIC_API_KEY or Options.APIKey")
	}

	var hints []string
	if opts.Language != "" {
		hints = append(hints, "The author says the language is "+opts.Language+".")
	}
	if opts.Context != "" {
		hints = append(hints, "Context from the author: "+opts.Context)
	}
	var hintBlock string
	if len(hints) > 0 {
		hintBlock = strings.Join(hints, "\n") + "\n\n"
	}
	userMsg := fmt.Sprintf("%sReview the following code:\n\n```\n%s\n```", hintBlock, code)

	reqBody, err := json.Marshal(apiRequest{
		Model:     opts.model(),
		MaxTokens: 2048,
		System:    systemPrompt,
		Tools: []apiTool{{
			Name:        "submit_review",
			Description: "Return a structured code review.",
			InputSchema: json.RawMessage(reviewToolSchema),
		}},
		ToolChoice: map[string]string{"type": "tool", "name": "submit_review"},
		Messages:   []apiMessage{{Role: "user", Content: userMsg}},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", key)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := opts.client().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	var parsed apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("anthropic API error: %s", parsed.Error.Message)
	}

	for _, block := range parsed.Content {
		if block.Type == "tool_use" && block.Name == "submit_review" {
			var result Result
			if err := json.Unmarshal(block.Input, &result); err != nil {
				return nil, fmt.Errorf("decode review: %w", err)
			}
			result.Model = opts.model()
			return &result, nil
		}
	}
	return nil, fmt.Errorf("model did not return a structured review (status %s)", resp.Status)
}

var severityIcon = map[string]string{"high": "🔴", "medium": "🟠", "low": "🟡"}

// ToMarkdown renders a review as a portable Markdown report.
func ToMarkdown(r *Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Smart Code Review\n\n")
	fmt.Fprintf(&b, "**Language:** %s  |  **Score:** %d/10  |  **Model:** %s\n\n",
		r.Language, r.OverallScore, r.Model)
	fmt.Fprintf(&b, "> %s\n\n", r.Summary)
	fmt.Fprintf(&b, "## Top 3 Improvements\n\n")
	for i, imp := range r.Improvements {
		icon := severityIcon[imp.Severity]
		if icon == "" {
			icon = "🟡"
		}
		loc := ""
		if imp.Location != "" {
			loc = fmt.Sprintf(" _(at %s)_", imp.Location)
		}
		fmt.Fprintf(&b, "### %d. %s %s `%s`%s\n\n", i+1, icon, imp.Title, imp.Category, loc)
		fmt.Fprintf(&b, "%s\n\n", imp.Explanation)
		fmt.Fprintf(&b, "**Suggestion:** %s\n\n", imp.Suggestion)
	}
	fmt.Fprintf(&b, "## ✅ What's good\n\n%s\n", r.PositiveNote)
	if len(r.ExtraObservations) > 0 {
		fmt.Fprintf(&b, "\n## Minor notes\n\n")
		for _, o := range r.ExtraObservations {
			fmt.Fprintf(&b, "- %s\n", o)
		}
	}
	return b.String()
}
