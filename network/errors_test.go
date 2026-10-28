package network

import (
	"errors"
	"testing"
)

func TestAPIError_ServerCodeFormat(t *testing.T) {
	err := newServerCodeError("/auth/login", 40001, "账号不存在")
	if err.Code != 40001 {
		t.Errorf("expected Code=40001, got %d", err.Code)
	}
	// 服务器业务错误必须保持 "<action> failed: <msg>" 格式以兼容既有解析逻辑。
	want := "/auth/login failed: 账号不存在"
	if err.Error() != want {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
	// 服务器业务错误默认不可重试。
	if IsRetryable(err) {
		t.Errorf("server code error should not be retryable")
	}
	if got := err.RetrySuggestion(); got != "" {
		t.Errorf("expected empty retry suggestion for server code error, got %q", got)
	}
}

func TestAPIError_ServerCodeEmptyMsg(t *testing.T) {
	err := newServerCodeError("/banks", 40002, "")
	if err.Error() != "/banks failed: 服务器返回错误" {
		t.Errorf("expected fallback message, got %q", err.Error())
	}
}

func TestAPIError_NetworkRetryable(t *testing.T) {
	err := newNetworkError("/auth/login", errors.New("connection refused"))
	if !IsRetryable(err) {
		t.Errorf("network error should be retryable")
	}
	if err.HTTPStatus != 0 || err.Code != 0 {
		t.Errorf("network error should have zero HTTPStatus and Code, got http=%d code=%d", err.HTTPStatus, err.Code)
	}
	if got := err.RetrySuggestion(); got == "" {
		t.Errorf("expected non-empty retry suggestion for network error")
	}
}

func TestAPIError_HTTPStatusRetryable(t *testing.T) {
	cases := []struct {
		status int
		retry  bool
	}{
		{500, true},
		{503, true},
		{429, true},
		{408, true},
		{401, false}, // RetrySuggestion 会给出"重新登录"，但 IsRetryable 仅对 5xx/429/408 置为 true
		{403, false},
		{404, false},
	}
	for _, c := range cases {
		err := newHTTPStatusError("/banks", c.status, "")
		if IsRetryable(err) != c.retry {
			t.Errorf("status %d: expected retryable=%v, got %v", c.status, c.retry, err.Retryable)
		}
		if err.HTTPStatus != c.status {
			t.Errorf("status %d: HTTPStatus field mismatch", c.status)
		}
	}
}

func TestAPIError_HTTPStatusMessage(t *testing.T) {
	err := newHTTPStatusError("/banks", 503, "service unavailable")
	want := "/banks: HTTP 503: service unavailable"
	if err.Error() != want {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
	// 无 body 时退化为纯状态码。
	err2 := newHTTPStatusError("/banks", 500, "")
	if err2.Error() != "/banks: HTTP 500" {
		t.Errorf("expected '/banks: HTTP 500', got %q", err2.Error())
	}
}

func TestAPIError_ParseErrorNotRetryable(t *testing.T) {
	err := newParseError("/banks", errors.New("invalid character 'x'"))
	if IsRetryable(err) {
		t.Errorf("parse error should not be retryable")
	}
	if err.HTTPStatus != 0 || err.Code != 0 {
		t.Errorf("parse error should have zero HTTPStatus and Code")
	}
}

func TestIsRetryable_NonAPIError(t *testing.T) {
	if IsRetryable(errors.New("plain error")) {
		t.Errorf("plain error should not be retryable")
	}
	var nilErr error
	if IsRetryable(nilErr) {
		t.Errorf("nil error should not be retryable")
	}
}

func TestAPIError_RetrySuggestionMapping(t *testing.T) {
	// RetrySuggestion 仅在 Retryable=true 时返回文案；401/403 虽在 RetrySuggestion 中单独处理，
	// 但 newHTTPStatusError 未将其标记为可重试，故返回空串——这是预期行为。
	cases := []struct {
		status int
		want   string
	}{
		{500, "服务器暂时异常，请稍后重试"},
		{429, "请求过于频繁，请稍后再试"},
		{408, "网络异常，请稍后重试"},
		{502, "服务器暂时异常，请稍后重试"},
		{401, ""}, // 不可重试（由调用方决定重新登录）
		{403, ""},
		{404, ""},
	}
	for _, c := range cases {
		err := newHTTPStatusError("/x", c.status, "")
		if got := err.RetrySuggestion(); got != c.want {
			t.Errorf("status %d: expected %q, got %q", c.status, c.want, got)
		}
	}
}
