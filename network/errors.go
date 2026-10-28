package network

import (
	"encoding/json"
	"errors"
	"fmt"
)

// APIError 统一的 API 错误类型。
//
// 所有通过 network 层发出的调用失败时，都应返回 *APIError（或可被 errors.As 提取的错误），
// 以便上层（core.HandleAPIError、UI 层）能够统一地：
//   - 区分"服务器业务错误(code!=0)"与"网络/HTTP/解析等基础设施错误";
//   - 获取人类可读的错误消息;
//   - 判断该错误是否可重试(retryable)，并给出相应的重试建议。
//
// discriminator（判别标志）：
//   - Code != 0            → 服务器业务错误（默认不可重试，Error() 保持 "<action> failed: <msg>" 格式以兼容既有解析逻辑）
//   - HTTPStatus != 0      → HTTP 状态码错误（5xx/429/408 可重试）
//   - 二者均为 0 且 Retryable=true → 网络层错误（可重试）
//   - 二者均为 0 且 Retryable=false    → 请求体/响应体解析错误（不可重试）
//
// 服务器业务错误的 Error() 输出严格保持 "<action> failed: <serverMsg>" 格式，
// 以兼容既有 core.hasServerErrorCode / handleServerErrorCode 的字符串解析逻辑。
type APIError struct {
	// Action 是触发错误的操作名称（英文小写，便于日志与重试建议复用），如 "login"、"fetch banks list"。
	Action string
	// Code 是服务器返回的业务错误码；非服务器业务错误时为 0。
	Code int
	// HTTPStatus 是 HTTP 状态码；非 HTTP 层错误（网络/解析）时为 0。
	HTTPStatus int
	// Message 是人类可读的错误消息（最终展示给用户的文本）。
	Message string
	// Retryable 标记该错误是否值得重试（网络抖动、5xx 服务器错误等）。
	Retryable bool
	// Endpoint 是触发错误的请求路径，便于日志定位。
	Endpoint string
}

// Error 实现 error 接口。
// 服务器业务错误保持 "<action> failed: <message>" 格式以兼容既有解析逻辑；
// 基础设施错误使用 "<action>: <message>" 格式。
func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("%s failed: %s", e.Action, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Action, e.Message)
}

// RetrySuggestion 返回针对该错误的中文重试建议文案。
// 对于不可重试的错误返回空字符串，调用方可据此决定是否展示"重试"按钮。
func (e *APIError) RetrySuggestion() string {
	if !e.Retryable {
		return ""
	}
	switch {
	case e.HTTPStatus >= 500:
		return "服务器暂时异常，请稍后重试"
	case e.HTTPStatus == 429:
		return "请求过于频繁，请稍后再试"
	case e.HTTPStatus == 401 || e.HTTPStatus == 403:
		return "认证失败，请重新登录"
	default:
		return "网络异常，请稍后重试"
	}
}

// IsRetryable 便捷方法：判断错误是否可重试（兼容非 *APIError 类型时退化为 false）。
func IsRetryable(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Retryable
	}
	return false
}

// newServerCodeError 构造服务器业务错误（code != 0）。
// action 为操作名称，code 为服务器返回的业务错误码，serverMsg 为服务器返回的具体消息。
func newServerCodeError(action string, code int, serverMsg string) *APIError {
	msg := serverMsg
	if msg == "" {
		msg = "服务器返回错误"
	}
	return &APIError{
		Action:    action,
		Code:      code,
		Message:   msg,
		Retryable: false,
	}
}

// newNetworkError 构造网络层错误（请求未成功发出或超时）。
func newNetworkError(action string, err error) *APIError {
	return &APIError{
		Action:    action,
		Message:   err.Error(),
		Retryable: true,
	}
}

// newHTTPStatusError 构造非 2xx HTTP 状态码错误。
func newHTTPStatusError(action string, httpStatus int, body string) *APIError {
	msg := fmt.Sprintf("HTTP %d", httpStatus)
	if body != "" {
		msg = fmt.Sprintf("HTTP %d: %s", httpStatus, body)
	}
	retryable := httpStatus >= 500 || httpStatus == 408 || httpStatus == 429
	return &APIError{
		Action:     action,
		HTTPStatus: httpStatus,
		Message:    msg,
		Retryable:  retryable,
	}
}

// newParseError 构造响应体解析错误。
func newParseError(action string, err error) *APIError {
	return &APIError{
		Action:    action,
		Message:   err.Error(),
		Retryable: false,
	}
}

// checkNon2xx 统一处理非 2xx HTTP 响应的错误判断逻辑。
//
// 消除多处重复的 `if resp.StatusCode < 200 || resp.StatusCode >= 300` 分支：
// 当 statusCode 落在 2xx 范围时返回 nil（调用方继续正常解析）；否则尝试从 body 中
// 提取服务器返回的 {code,msg} 业务字段，并包装为 *APIError（HTTPStatus 错误）。
//
// 调用方应直接使用其返回值作为 early return：
//
//	if err := checkNon2xx(endpoint, resp.StatusCode, bodyBytes); err != nil {
//		return nil, err
//	}
func checkNon2xx(endpoint string, statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	var apiErr struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.Unmarshal(body, &apiErr)
	return newHTTPStatusError(endpoint, statusCode, apiErr.Msg)
}
