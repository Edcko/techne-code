package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Edcko/techne-code/pkg/tool"
)

type WebFetchTool struct {
	Client         *http.Client
	MaxContentLen  int
	DefaultTimeout time.Duration
}

func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		MaxContentLen:  51200,
		DefaultTimeout: 30 * time.Second,
		Client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

type webFetchParams struct {
	URL           string `json:"url"`
	MaxContentLen int    `json:"max_content_length,omitempty"`
}

func (t *WebFetchTool) Name() string { return "web_fetch" }

func (t *WebFetchTool) Description() string {
	return "Fetch a URL and return its content as text. Supports HTTP and HTTPS URLs. HTML is converted to plain text."
}

func (t *WebFetchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {"type": "string", "description": "The URL to fetch (HTTP or HTTPS)"},
			"max_content_length": {"type": "integer", "description": "Maximum content length in bytes (default 51200, optional)"}
		},
		"required": ["url"]
	}`)
}

func (t *WebFetchTool) RequiresPermission() bool { return true }

func (t *WebFetchTool) Execute(ctx context.Context, input json.RawMessage) (tool.ToolResult, error) {
	var params webFetchParams
	if err := json.Unmarshal(input, &params); err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error parsing parameters: %v", err), IsError: true}, nil
	}

	if params.URL == "" {
		return tool.ToolResult{Content: "Error: url is required", IsError: true}, nil
	}

	if !strings.HasPrefix(params.URL, "http://") && !strings.HasPrefix(params.URL, "https://") {
		return tool.ToolResult{Content: "Error: only HTTP and HTTPS URLs are supported", IsError: true}, nil
	}

	maxLen := params.MaxContentLen
	if maxLen <= 0 {
		maxLen = t.MaxContentLen
	}

	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: t.DefaultTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error creating request: %v", err), IsError: true}, nil
	}

	req.Header.Set("User-Agent", "TechneCode/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error fetching URL: %v", err), IsError: true}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return tool.ToolResult{
			Content: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
			IsError: true,
		}, nil
	}

	contentType := resp.Header.Get("Content-Type")
	if isBinaryContentType(contentType) {
		return tool.ToolResult{
			Content: fmt.Sprintf("Error: binary content type %q is not supported", contentType),
			IsError: true,
		}, nil
	}

	limitedReader := io.LimitReader(resp.Body, int64(maxLen+1))
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return tool.ToolResult{Content: fmt.Sprintf("Error reading response: %v", err), IsError: true}, nil
	}

	if len(body) > maxLen {
		return tool.ToolResult{
			Content: fmt.Sprintf("Error: content exceeds maximum length of %d bytes", maxLen),
			IsError: true,
		}, nil
	}

	content := string(body)

	if isHTMLContent(contentType, content) {
		content = htmlToText(content)
	}

	return tool.ToolResult{Content: content}, nil
}

func isBinaryContentType(ct string) bool {
	ct = strings.ToLower(ct)
	binaryTypes := []string{
		"image/", "video/", "audio/", "application/pdf",
		"application/zip", "application/gzip", "application/x-tar",
		"application/octet-stream", "application/x-binary",
	}
	for _, bt := range binaryTypes {
		if strings.HasPrefix(ct, bt) {
			return true
		}
	}
	return false
}

func isHTMLContent(ct string, body string) bool {
	ct = strings.ToLower(ct)
	if strings.Contains(ct, "text/html") {
		return true
	}
	trimmed := strings.TrimSpace(body)
	return strings.HasPrefix(trimmed, "<!doctype html") ||
		strings.HasPrefix(trimmed, "<html") ||
		strings.HasPrefix(trimmed, "<HTML")
}

var (
	reStyle     = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reScript    = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reTags      = regexp.MustCompile(`<[^>]+>`)
	reEntities  = regexp.MustCompile(`&nbsp;`)
	reAmp       = regexp.MustCompile(`&amp;`)
	reLt        = regexp.MustCompile(`&lt;`)
	reGt        = regexp.MustCompile(`&gt;`)
	reQuote     = regexp.MustCompile(`&quot;`)
	reMultiLine = regexp.MustCompile(`\n{3,}`)
)

func htmlToText(html string) string {
	text := reStyle.ReplaceAllString(html, "")
	text = reScript.ReplaceAllString(text, "")

	text = reTags.ReplaceAllString(text, "")

	text = reEntities.ReplaceAllString(text, " ")
	text = reAmp.ReplaceAllString(text, "&")
	text = reLt.ReplaceAllString(text, "<")
	text = reGt.ReplaceAllString(text, ">")
	text = reQuote.ReplaceAllString(text, `"`)

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\t", "  ")
	text = reMultiLine.ReplaceAllString(text, "\n\n")
	text = strings.TrimSpace(text)

	return text
}
