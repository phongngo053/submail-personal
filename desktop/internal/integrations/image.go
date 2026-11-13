package integrations

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ImageRequest is the body sent to an OpenAI-compatible images endpoint.
type ImageRequest struct {
	Prompt         string `json:"prompt"`
	Model          string `json:"model,omitempty"`
	ResponseFormat string `json:"response_format"`
	N              int    `json:"n"`
	Size           string `json:"size,omitempty"`
}

// ImageData is one generated image inside the reply.
type ImageData struct {
	URL           string `json:"url"`
	B64JSON       string `json:"b64_json"`
	RevisedPrompt string `json:"revised_prompt"`
}

// ImageResponse is the reply of an OpenAI-compatible images endpoint. Both
// shapes are accepted: a remote URL and an inline base64 payload.
type ImageResponse struct {
	Data  []ImageData `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// ImageClient calls an OpenAI-compatible text-to-image service.
type ImageClient struct {
	client  *http.Client
	baseURL string
	apiKey  string
	model   string
}

// NewImageClient returns a client for the given integration.
func NewImageClient(it Integration) *ImageClient {
	return &ImageClient{
		client:  &http.Client{Timeout: 180 * time.Second},
		baseURL: strings.TrimRight(it.BaseURL, "/"),
		apiKey:  it.APIKey,
		model:   it.Model,
	}
}

// maxImageBytes caps the download so a broken response cannot fill the disk.
const maxImageBytes = 12 << 20

// Generate asks the service for one image and returns the PNG bytes.
func (c *ImageClient) Generate(ctx context.Context, prompt, size string) ([]byte, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, fmt.Errorf("integrations: prompt không được để trống")
	}
	if size == "" {
		size = "1024x1024"
	}

	body, err := json.Marshal(ImageRequest{
		Prompt:         prompt,
		Model:          c.model,
		ResponseFormat: "b64_json",
		N:              1,
		Size:           size,
	})
	if err != nil {
		return nil, fmt.Errorf("integrations: không mã hoá được yêu cầu: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("integrations: không tạo được yêu cầu: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("integrations: không gọi được dịch vụ tạo ảnh tại %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes))
	if err != nil {
		return nil, fmt.Errorf("integrations: không đọc được phản hồi tạo ảnh: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("integrations: dịch vụ tạo ảnh trả về mã %d: %s", resp.StatusCode, snippet(raw))
	}

	var parsed ImageResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("integrations: không đọc được phản hồi tạo ảnh: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("integrations: dịch vụ tạo ảnh báo lỗi: %s", parsed.Error.Message)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("integrations: dịch vụ tạo ảnh không trả về ảnh nào")
	}

	entry := parsed.Data[0]
	if entry.B64JSON != "" {
		data, err := base64.StdEncoding.DecodeString(entry.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("integrations: ảnh trả về không phải base64 hợp lệ: %w", err)
		}
		return data, nil
	}
	if entry.URL != "" {
		return c.download(ctx, entry.URL)
	}
	return nil, fmt.Errorf("integrations: dịch vụ tạo ảnh không trả về ảnh nào")
}

// download fetches a generated image and returns its bytes.
func (c *ImageClient) download(ctx context.Context, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("integrations: không tạo được yêu cầu tải ảnh: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("integrations: không tải được ảnh đã tạo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("integrations: tải ảnh thất bại với mã %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes))
	if err != nil {
		return nil, fmt.Errorf("integrations: không đọc được ảnh đã tạo: %w", err)
	}
	return data, nil
}

// snippet trims a response body so an error message stays readable.
func snippet(raw []byte) string {
	const limit = 200
	text := strings.TrimSpace(string(raw))
	if len(text) > limit {
		return text[:limit] + "..."
	}
	return text
}
