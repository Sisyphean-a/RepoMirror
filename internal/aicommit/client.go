package aicommit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	defaultBaseURL = "https://api.deepseek.com"
	defaultModel   = "deepseek-v4-flash"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

type requestBody struct {
	Model          string          `json:"model"`
	Messages       []message       `json:"messages"`
	ResponseFormat responseFormat  `json:"response_format"`
	Thinking       thinkingOptions `json:"thinking"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type thinkingOptions struct {
	Type string `json:"type"`
}

type responseBody struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type generatedMessage struct {
	Message string `json:"message"`
}

func NewClient() *Client {
	return &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{},
		model:      defaultModel,
	}
}

func (client *Client) Generate(apiKey string, changes string) (string, error) {
	payload, err := client.buildRequestBody(changes)
	if err != nil {
		return "", err
	}
	response, err := client.doRequest(apiKey, payload)
	if err != nil {
		return "", err
	}
	return parseGeneratedMessage(response)
}

func (client *Client) buildRequestBody(changes string) ([]byte, error) {
	body := requestBody{
		Model: defaultModel,
		Messages: []message{
			{Role: "system", Content: strings.TrimSpace(systemPrompt)},
			{Role: "user", Content: changes},
		},
		ResponseFormat: responseFormat{Type: "json_object"},
		Thinking:       thinkingOptions{Type: "disabled"},
	}
	return json.Marshal(body)
}

func (client *Client) doRequest(apiKey string, payload []byte) ([]byte, error) {
	request, err := http.NewRequest(http.MethodPost, client.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	request.Header.Set("Content-Type", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := readResponseBody(response)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("DeepSeek request failed: %s", strings.TrimSpace(string(body)))
	}
	return body, nil
}

func readResponseBody(response *http.Response) ([]byte, error) {
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(response.Body); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func parseGeneratedMessage(body []byte) (string, error) {
	var completion responseBody
	if err := json.Unmarshal(body, &completion); err != nil {
		return "", err
	}
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("DeepSeek returned no choices")
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("DeepSeek returned an empty message")
	}
	return extractMessage(content)
}

func extractMessage(content string) (string, error) {
	var message generatedMessage
	if err := json.Unmarshal([]byte(content), &message); err != nil {
		return "", err
	}
	if strings.TrimSpace(message.Message) == "" {
		return "", fmt.Errorf("DeepSeek returned an empty commit message")
	}
	return strings.TrimSpace(message.Message), nil
}
