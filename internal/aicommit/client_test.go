package aicommit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientGenerateBuildsDeepSeekRequest(t *testing.T) {
	var seenAuth string
	var seenBody requestBody
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		seenAuth = request.Header.Get("Authorization")
		payload, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read request body failed: %v", err)
		}
		if err := json.Unmarshal(payload, &seenBody); err != nil {
			t.Fatalf("unmarshal request failed: %v", err)
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"{\"message\":\"feat(sync): 增加 AI 生成提交信息\"}"}}]}`))
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	message, err := client.Generate("test-key", "Git status:\n M tracked.txt")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if message != "feat(sync): 增加 AI 生成提交信息" {
		t.Fatalf("unexpected message: %s", message)
	}
	if seenAuth != "Bearer test-key" {
		t.Fatalf("unexpected auth header: %s", seenAuth)
	}
	if seenBody.Model != defaultModel {
		t.Fatalf("unexpected model: %s", seenBody.Model)
	}
	if seenBody.ResponseFormat.Type != "json_object" {
		t.Fatalf("unexpected response format: %+v", seenBody.ResponseFormat)
	}
	if seenBody.Thinking.Type != "disabled" {
		t.Fatalf("unexpected thinking config: %+v", seenBody.Thinking)
	}
	if len(seenBody.Messages) != 2 {
		t.Fatalf("unexpected message count: %d", len(seenBody.Messages))
	}
	if !strings.Contains(seenBody.Messages[1].Content, "tracked.txt") {
		t.Fatalf("request should include change summary, got %q", seenBody.Messages[1].Content)
	}
}

func TestParseGeneratedMessageRejectsEmptyPayload(t *testing.T) {
	_, err := parseGeneratedMessage([]byte(`{"choices":[{"message":{"content":"{\"message\":\"   \"}"}}]}`))
	if err == nil {
		t.Fatal("expected parseGeneratedMessage to reject an empty commit message")
	}
}
