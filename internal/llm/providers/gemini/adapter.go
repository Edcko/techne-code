package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Edcko/techne-code/pkg/provider"
)

type Adapter struct {
	apiKey  string
	baseURL string
	client  *http.Client
	models  []provider.ModelInfo
}

func NewAdapter(apiKey, baseURL string, models []provider.ModelInfo) *Adapter {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	if len(models) == 0 {
		models = []provider.ModelInfo{
			{ID: "gemini-2.5-pro", MaxTokens: 65536, SupportsTools: true, SupportsVision: true, ContextWindow: 1048576},
			{ID: "gemini-2.5-flash", MaxTokens: 65536, SupportsTools: true, SupportsVision: true, ContextWindow: 1048576},
			{ID: "gemini-2.0-flash", MaxTokens: 8192, SupportsTools: true, SupportsVision: true, ContextWindow: 1048576},
		}
	}

	return &Adapter{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Minute,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 5 * time.Minute,
			},
		},
		models: models,
	}
}

func (a *Adapter) Name() string {
	return "gemini"
}

func (a *Adapter) Models() []provider.ModelInfo {
	return a.models
}

func (a *Adapter) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	body := a.buildRequest(req)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", a.baseURL, url.PathEscape(req.Config.Model), url.QueryEscape(a.apiKey))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, convertError(err, 0)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, convertError(fmt.Errorf("API error: %s", string(bodyBytes)), resp.StatusCode)
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return convertResponse(&genResp), nil
}

func (a *Adapter) Stream(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamChunk, error) {
	body := a.buildRequest(req)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", a.baseURL, url.PathEscape(req.Config.Model), url.QueryEscape(a.apiKey))
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, convertError(err, 0)
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, convertError(fmt.Errorf("API error: %s", string(bodyBytes)), resp.StatusCode)
	}

	ch := make(chan provider.StreamChunk, 100)

	go func() {
		defer close(ch)
		defer resp.Body.Close()
		a.processStream(resp.Body, ch)
	}()

	return ch, nil
}

func (a *Adapter) buildRequest(req provider.ChatRequest) generateRequest {
	body := generateRequest{
		Contents: convertMessages(req.Messages),
		GenerationConfig: &generationConfig{
			MaxOutputTokens: req.Config.MaxTokens,
			Temperature:     req.Config.Temperature,
			TopP:            req.Config.TopP,
		},
	}

	if req.System != "" {
		body.SystemInstruction = &content{
			Parts: []part{{Text: req.System}},
		}
	}

	if len(req.Tools) > 0 {
		body.Tools = convertTools(req.Tools)
	}

	return body
}

func (a *Adapter) processStream(reader io.Reader, ch chan<- provider.StreamChunk) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var chunk generateResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Candidates) == 0 || chunk.Candidates[0].Content == nil {
			continue
		}

		cand := chunk.Candidates[0]

		for _, p := range cand.Content.Parts {
			if p.Text != "" {
				ch <- provider.StreamChunk{
					Type: "text_delta",
					Text: p.Text,
				}
			}
			if p.FunctionCall != nil {
				argsJSON := "{}"
				if len(p.FunctionCall.Args) > 0 {
					argsJSON = string(p.FunctionCall.Args)
				}
				ch <- provider.StreamChunk{
					Type: "tool_call_delta",
					ToolCall: &provider.ToolCallDelta{
						ID:        generateCallID(),
						Name:      p.FunctionCall.Name,
						InputJSON: argsJSON,
						Done:      true,
					},
				}
			}
		}

		if cand.FinishReason != "" {
			ch <- provider.StreamChunk{Type: "done"}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- provider.StreamChunk{
			Type:  "error",
			Error: convertError(err, 0),
		}
	}
}
