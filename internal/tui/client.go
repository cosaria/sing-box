package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StatusResp 是 GET /api/status 的响应结构。
type StatusResp struct {
	Engine   string `json:"engine"`
	Inbounds int    `json:"inbounds"`
}

// Inbound 表示一条入站规则。
type Inbound struct {
	ID        int64  `json:"id"`
	Tag       string `json:"tag"`
	Protocol  string `json:"protocol"`
	Port      uint16 `json:"port"`
	Settings  string `json:"settings"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// TrafficSummary 是单个入站标签的流量汇总。
type TrafficSummary struct {
	Tag      string `json:"tag"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

// Client 封装对管理 HTTP API 的所有调用。
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient 创建新的 API 客户端。
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// do 发送带认证 header 的 HTTP 请求。
func (c *Client) do(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

// decode 是泛型 JSON 解码 helper，关闭响应体。
func decode[T any](resp *http.Response) (T, error) {
	defer resp.Body.Close()
	var zero T
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return zero, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return zero, fmt.Errorf("failed to decode response: %w", err)
	}
	return v, nil
}

// Status 获取引擎运行状态。
func (c *Client) Status() (*StatusResp, error) {
	resp, err := c.do(http.MethodGet, "/api/status", nil)
	if err != nil {
		return nil, err
	}
	v, err := decode[StatusResp](resp)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListInbounds 列出所有入站规则。
func (c *Client) ListInbounds() ([]Inbound, error) {
	resp, err := c.do(http.MethodGet, "/api/inbounds", nil)
	if err != nil {
		return nil, err
	}
	return decode[[]Inbound](resp)
}

// CreateInbound 创建新的入站规则。
func (c *Client) CreateInbound(protocol string, port uint16) (*Inbound, error) {
	payload := map[string]any{
		"protocol": protocol,
		"port":     port,
	}
	resp, err := c.do(http.MethodPost, "/api/inbounds", payload)
	if err != nil {
		return nil, err
	}
	v, err := decode[Inbound](resp)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetInbound 根据 ID 获取单个入站规则。
func (c *Client) GetInbound(id int64) (*Inbound, error) {
	resp, err := c.do(http.MethodGet, fmt.Sprintf("/api/inbounds/%d", id), nil)
	if err != nil {
		return nil, err
	}
	v, err := decode[Inbound](resp)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// UpdateInbound 更新指定入站规则的端口或配置。
func (c *Client) UpdateInbound(id int64, port *uint16, settings *json.RawMessage) (*Inbound, error) {
	payload := map[string]any{}
	if port != nil {
		payload["port"] = *port
	}
	if settings != nil {
		payload["settings"] = settings
	}
	resp, err := c.do(http.MethodPut, fmt.Sprintf("/api/inbounds/%d", id), payload)
	if err != nil {
		return nil, err
	}
	v, err := decode[Inbound](resp)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// DeleteInbound 删除指定入站规则。
func (c *Client) DeleteInbound(id int64) error {
	resp, err := c.do(http.MethodDelete, fmt.Sprintf("/api/inbounds/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetStats 获取所有入站的流量汇总。
func (c *Client) GetStats() ([]TrafficSummary, error) {
	resp, err := c.do(http.MethodGet, "/api/stats", nil)
	if err != nil {
		return nil, err
	}
	return decode[[]TrafficSummary](resp)
}

// Reload 触发引擎重新加载配置。
func (c *Client) Reload() error {
	resp, err := c.do(http.MethodPost, "/api/reload", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reload failed with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
