package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/teal-bauer/specter/internal/config"
)

// Client is a Ghost Admin API client
type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

// NewClient creates a new Ghost Admin API client from config
func NewClient(cfg *config.Config) *Client {
	baseURL := strings.TrimSuffix(cfg.URL, "/")
	return &Client{
		baseURL: baseURL,
		key:     cfg.Key,
		http:    &http.Client{},
	}
}

// APIError represents an error from the Ghost API
type APIError struct {
	Errors []struct {
		Message string `json:"message"`
		Context string `json:"context,omitempty"`
		Type    string `json:"type,omitempty"`
	} `json:"errors"`
}

func (e *APIError) Error() string {
	if len(e.Errors) == 0 {
		return "unknown API error"
	}
	msg := e.Errors[0].Message
	if e.Errors[0].Context != "" {
		msg += ": " + e.Errors[0].Context
	}
	return msg
}

func (c *Client) apiURL(path string) string {
	return c.baseURL + "/ghost/api/admin" + path
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	token, err := GenerateToken(c.key)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.apiURL(path), reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Ghost "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept-Version", "v5.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && len(apiErr.Errors) > 0 {
			return nil, &apiErr
		}
		return nil, fmt.Errorf("API error: %s (status %d)", string(respBody), resp.StatusCode)
	}

	return respBody, nil
}

// Get performs a GET request
func (c *Client) Get(path string, params url.Values) ([]byte, error) {
	fullPath := path
	if len(params) > 0 {
		fullPath += "?" + params.Encode()
	}
	return c.doRequest("GET", fullPath, nil)
}

// Post performs a POST request
func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	return c.doRequest("POST", path, body)
}

// Put performs a PUT request
func (c *Client) Put(path string, body interface{}) ([]byte, error) {
	return c.doRequest("PUT", path, body)
}

// Delete performs a DELETE request
func (c *Client) Delete(path string) ([]byte, error) {
	return c.doRequest("DELETE", path, nil)
}

// UploadImage uploads an image file to Ghost
func (c *Client) UploadImage(filePath, ref string) (string, error) {
	token, err := GenerateToken(c.key)
	if err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("creating form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("copying file: %w", err)
	}

	if ref != "" {
		if err := writer.WriteField("ref", ref); err != nil {
			return "", fmt.Errorf("writing ref field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("closing writer: %w", err)
	}

	req, err := http.NewRequest("POST", c.apiURL("/images/upload/"), body)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Ghost "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept-Version", "v5.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if err := json.Unmarshal(respBody, &apiErr); err == nil && len(apiErr.Errors) > 0 {
			return "", &apiErr
		}
		return "", fmt.Errorf("upload error: %s", string(respBody))
	}

	var result struct {
		Images []struct {
			URL string `json:"url"`
			Ref string `json:"ref,omitempty"`
		} `json:"images"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Images) == 0 {
		return "", fmt.Errorf("no image URL in response")
	}

	return result.Images[0].URL, nil
}
