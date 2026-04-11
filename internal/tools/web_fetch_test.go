package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebFetchTool_Name(t *testing.T) {
	tool := NewWebFetchTool()
	if tool.Name() != "web_fetch" {
		t.Errorf("expected name 'web_fetch', got %q", tool.Name())
	}
}

func TestWebFetchTool_Description(t *testing.T) {
	tool := NewWebFetchTool()
	if tool.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestWebFetchTool_Parameters(t *testing.T) {
	tool := NewWebFetchTool()
	params := tool.Parameters()
	if !json.Valid(params) {
		t.Error("parameters should be valid JSON")
	}
}

func TestWebFetchTool_RequiresPermission(t *testing.T) {
	tool := NewWebFetchTool()
	if !tool.RequiresPermission() {
		t.Error("web_fetch tool should require permission")
	}
}

func TestWebFetchTool_Execute_SuccessfulFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from server"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "hello from server") {
		t.Errorf("expected content to contain 'hello from server', got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_HTMLToText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body><h1>Title</h1><p>Hello World</p></body></html>"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if strings.Contains(result.Content, "<h1>") {
		t.Errorf("HTML tags should be stripped, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Title") {
		t.Errorf("expected 'Title' in output, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Hello World") {
		t.Errorf("expected 'Hello World' in output, got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_404Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for 404 response")
	}
	if !strings.Contains(result.Content, "404") {
		t.Errorf("expected '404' in error, got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Write([]byte("too slow"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	ft.Client.Timeout = 100 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	input, _ := json.Marshal(map[string]string{"url": server.URL})
	result, err := ft.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for timeout")
	}
}

func TestWebFetchTool_Execute_BinaryRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("not a real pdf"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for binary content type")
	}
	if !strings.Contains(result.Content, "binary") {
		t.Errorf("expected 'binary' in error, got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_ImageRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("not a real image"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for image content type")
	}
}

func TestWebFetchTool_Execute_MaxContentLength(t *testing.T) {
	bigContent := strings.Repeat("a", 60000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(bigContent))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]interface{}{
		"url":                server.URL,
		"max_content_length": 1000,
	})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for content exceeding max length")
	}
	if !strings.Contains(result.Content, "exceeds") {
		t.Errorf("expected 'exceeds' in error, got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_InvalidScheme(t *testing.T) {
	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": "ftp://example.com/file"})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for non-HTTP URL")
	}
	if !strings.Contains(result.Content, "HTTP") {
		t.Errorf("expected 'HTTP' in error, got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_EmptyURL(t *testing.T) {
	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": ""})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for empty URL")
	}
}

func TestWebFetchTool_Execute_MissingURL(t *testing.T) {
	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for missing URL")
	}
}

func TestWebFetchTool_Execute_InvalidJSON(t *testing.T) {
	ft := NewWebFetchTool()
	result, err := ft.Execute(context.Background(), json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid JSON")
	}
}

func TestWebFetchTool_Execute_FollowsRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/old" {
			http.Redirect(w, r, "/new", http.StatusMovedPermanently)
			return
		}
		w.Write([]byte("redirected successfully"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL + "/old"})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "redirected successfully") {
		t.Errorf("expected 'redirected successfully', got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_StripScriptsAndStyles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><style>body{color:red}</style><script>alert("xss")</script></head><body><p>Content</p></body></html>`))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if strings.Contains(result.Content, "alert") {
		t.Errorf("script content should be stripped, got: %s", result.Content)
	}
	if strings.Contains(result.Content, "color:red") {
		t.Errorf("style content should be stripped, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "Content") {
		t.Errorf("expected 'Content' in output, got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_500Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for 500 response")
	}
	if !strings.Contains(result.Content, "500") {
		t.Errorf("expected '500' in error, got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_PlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("plain text response"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL})
	result, err := ft.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %s", result.Content)
	}
	if result.Content != "plain text response" {
		t.Errorf("expected 'plain text response', got: %s", result.Content)
	}
}

func TestWebFetchTool_Execute_UserAgent(t *testing.T) {
	var userAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent = r.Header.Get("User-Agent")
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	ft := NewWebFetchTool()
	input, _ := json.Marshal(map[string]string{"url": server.URL})
	ft.Execute(context.Background(), input)

	if !strings.Contains(userAgent, "TechneCode") {
		t.Errorf("expected User-Agent to contain 'TechneCode', got: %s", userAgent)
	}
}

func TestHTMLToText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		omits    string
	}{
		{
			name:     "strips tags",
			input:    "<p>Hello</p>",
			contains: "Hello",
			omits:    "<p>",
		},
		{
			name:     "decodes entities",
			input:    "a &amp; b &lt; c",
			contains: "a & b < c",
			omits:    "&amp;",
		},
		{
			name:     "strips script",
			input:    "<script>evil()</script>safe",
			contains: "safe",
			omits:    "evil",
		},
		{
			name:     "strips style",
			input:    "<style>body{}</style>text",
			contains: "text",
			omits:    "body{}",
		},
		{
			name:     "nbsp entity",
			input:    "hello&nbsp;world",
			contains: "hello",
			omits:    "&nbsp;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := htmlToText(tt.input)
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q, got: %s", tt.contains, result)
			}
			if tt.omits != "" && strings.Contains(result, tt.omits) {
				t.Errorf("expected result to omit %q, got: %s", tt.omits, result)
			}
		})
	}
}

func TestIsBinaryContentType(t *testing.T) {
	tests := []struct {
		ct       string
		expected bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"video/mp4", true},
		{"audio/mpeg", true},
		{"application/pdf", true},
		{"application/zip", true},
		{"application/octet-stream", true},
		{"text/html", false},
		{"text/plain", false},
		{"application/json", false},
		{"application/xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.ct, func(t *testing.T) {
			result := isBinaryContentType(tt.ct)
			if result != tt.expected {
				t.Errorf("isBinaryContentType(%q) = %v, want %v", tt.ct, result, tt.expected)
			}
		})
	}
}
