package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gzjjjfree/practice/parser"
)

// doRequest performs an HTTP request with JSON body and returns parsed response using context.Background().
func (c *APIClient) doRequest(method, endpoint string, reqBody, respBody interface{}) error {
	return c.doRequestWithContext(context.Background(), method, endpoint, reqBody, respBody)
}

// doRequestWithContext performs an HTTP request with context, JSON body and returns parsed response.
// Accepts all 2xx success status codes (200 OK, 201 Created, etc.).
// Includes retry mechanism for transient network or server errors, tiered LRU caching, request deduplication, and streaming JSON decoding.
func (c *APIClient) doRequestWithContext(ctx context.Context, method, endpoint string, reqBody, respBody interface{}) error {
	var reqBodyBytes []byte
	var bodyBytes []byte
	var err error

	if reqBody != nil {
		reqBodyBytes, err = json.Marshal(reqBody)
		if err != nil {
			return newParseError("request body", err)
		}
	}

	cacheKey := fmt.Sprintf("%s:%s:%s", method, endpoint, string(reqBodyBytes))

	// 6.2.1 区分读写缓存策略：题库/模板等静态数据可缓存，用户信息/考试等动态数据不缓存
	shouldCache := method == "GET" && (strings.Contains(endpoint, "/banks") || strings.Contains(endpoint, "/templates"))

	if shouldCache {
		if cached, ok := c.cache.get(cacheKey); ok {
			if respBody != nil {
				decoder := json.NewDecoder(bytes.NewReader(cached))
				return decoder.Decode(respBody)
			}
			return nil
		}

		c.inFlightMu.Lock()
		if c.inFlight == nil {
			c.inFlight = make(map[string]*inFlightCall)
		}
		if call, exists := c.inFlight[cacheKey]; exists {
			c.inFlightMu.Unlock()
			call.wg.Wait()
			if call.err != nil {
				return call.err
			}
			if respBody != nil && len(call.val) > 0 {
				decoder := json.NewDecoder(bytes.NewReader(call.val))
				return decoder.Decode(respBody)
			}
			return nil
		}
		call := &inFlightCall{}
		call.wg.Add(1)
		c.inFlight[cacheKey] = call
		c.inFlightMu.Unlock()

		defer func() {
			c.inFlightMu.Lock()
			delete(c.inFlight, cacheKey)
			call.err = err
			if err == nil && len(bodyBytes) > 0 {
				call.val = bodyBytes
			}
			c.inFlightMu.Unlock()
			call.wg.Done()
		}()
	} else {
		// 非缓存请求或非 GET 请求：清理缓存（6.2.2 缓存失效机制）
		if method != "GET" && c.cache != nil {
			c.cache.clear()
		}
	}

	startTime := time.Now()
	maxRetries := 3
	var resp *http.Response

	for attempt := 1; attempt <= maxRetries; attempt++ {
		var req *http.Request
		if reqBodyBytes != nil {
			req, err = http.NewRequestWithContext(ctx, method, BaseURL+endpoint, bytes.NewReader(reqBodyBytes))
			if err != nil {
				duration := time.Since(startTime)
				slog.Error("failed to create request", "endpoint", endpoint, "duration", duration, "err", err)
				return newParseError("request body", err)
			}
			req.Header.Set("Content-Type", "application/json")
		} else {
			req, err = http.NewRequestWithContext(ctx, method, BaseURL+endpoint, nil)
			if err != nil {
				duration := time.Since(startTime)
				slog.Error("failed to create request", "endpoint", endpoint, "duration", duration, "err", err)
				return newParseError("request body", err)
			}
		}

		// Always send Authorization header — server requires it for all requests (including login).
		// When token is empty, send empty Bearer to satisfy server's header check.
		req.Header.Set("Authorization", "Bearer "+c.token)

		resp, err = c.client.Do(req)
		if err != nil {
			if attempt < maxRetries {
				slog.Debug("network error, retrying", "endpoint", endpoint, "attempt", attempt, "maxRetries", maxRetries, "err", err)
				base := 300 * time.Millisecond
				backoff := base * time.Duration(1<<uint(attempt-1))
				if backoff > 4*time.Second {
					backoff = 4 * time.Second
				}
				jitter := time.Duration(time.Now().UnixNano()%200-100) * time.Millisecond
				sleepDuration := backoff + jitter
				if sleepDuration < 100*time.Millisecond {
					sleepDuration = 100 * time.Millisecond
				}
				time.Sleep(sleepDuration)
				continue
			}
			duration := time.Since(startTime)
			slog.Error("network error after retries", "endpoint", endpoint, "duration", duration, "err", err)
			return newNetworkError(endpoint, err)
		}

		// Handle 401 Unauthorized token auto-refresh
		if resp.StatusCode == 401 && endpoint != "/auth/login" && endpoint != "/auth/refresh" {
			resp.Body.Close()
			storage := NewTokenStorage()
			refreshToken := storage.LoadRefreshToken()
			if refreshToken != "" {
				if _, refErr := c.RefreshToken(refreshToken); refErr == nil {
					storage.SaveToken(c.token)
					// Retry request with new token
					if reqBodyBytes != nil {
						req, err = http.NewRequestWithContext(ctx, method, BaseURL+endpoint, bytes.NewReader(reqBodyBytes))
					} else {
						req, err = http.NewRequestWithContext(ctx, method, BaseURL+endpoint, nil)
					}
					if err == nil {
						req.Header.Set("Content-Type", "application/json")
						req.Header.Set("Authorization", "Bearer "+c.token)
						resp, err = c.client.Do(req)
						if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
							break
						}
					}
				}
			}
			storage.ClearToken()
			c.token = ""
		}

		// Retry on 5xx server errors
		if resp.StatusCode >= 500 && attempt < maxRetries {
			resp.Body.Close()
			slog.Debug("server error, retrying", "endpoint", endpoint, "status", resp.StatusCode, "attempt", attempt)
			base := 300 * time.Millisecond
			backoff := base * time.Duration(1<<uint(attempt-1))
			if backoff > 4*time.Second {
				backoff = 4 * time.Second
			}
			jitter := time.Duration(time.Now().UnixNano()%200-100) * time.Millisecond
			sleepDuration := backoff + jitter
			if sleepDuration < 100*time.Millisecond {
				sleepDuration = 100 * time.Millisecond
			}
			time.Sleep(sleepDuration)
			continue
		}

		break
	}
	defer resp.Body.Close()
	duration := time.Since(startTime)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("HTTP request succeeded", "endpoint", endpoint, "status", resp.StatusCode, "duration", duration)
	} else {
		slog.Warn("HTTP request returned non-2xx status", "endpoint", endpoint, "status", resp.StatusCode, "duration", duration)
	}

	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return newParseError(endpoint, err)
	}
	bodyBytes = parser.FixEncodingBytes(bodyBytes)

	// Accept all 2xx success codes (200 OK, 201 Created, etc.).
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if shouldCache && c.cache != nil {
			c.cache.put(cacheKey, bodyBytes)
			// Also update inFlight call.val if it exists
			c.inFlightMu.Lock()
			if call, exists := c.inFlight[cacheKey]; exists {
				call.val = bodyBytes
			}
			c.inFlightMu.Unlock()
		}
		if respBody != nil {
			decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
			if err := decoder.Decode(respBody); err != nil {
				return newParseError(endpoint, err)
			}
		}
		return nil
	}

	// For non-2xx status codes, try to parse JSON response body for {code, msg, error}.
	bodyStr := string(bodyBytes)
	fmt.Printf("[API_DOREQUEST] non-2xx status=%d, body=%q\n", resp.StatusCode, bodyStr)

	var apiErr struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &apiErr); err == nil {
		// 根据 error 字段生成具体的错误信息
		errMsg := apiErr.Msg
		if apiErr.Error != "" {
			switch {
			case apiErr.Error == "user not found":
				errMsg = "账号不存在"
			case apiErr.Error == "wrong password":
				errMsg = "密码错误"
			case strings.Contains(apiErr.Error, "exam has timeout"):
				errMsg = "考试已超时"
			default:
				errMsg = apiErr.Error
			}
		}

		// Also try to parse into respBody if it's a wrapped response
		if respBody != nil {
			var wrappedResp struct {
				Code int         `json:"code"`
				Msg  string      `json:"msg"`
				Data interface{} `json:"data"`
			}
			if err2 := json.Unmarshal(bodyBytes, &wrappedResp); err2 == nil {
				wrappedResp.Code = apiErr.Code
				wrappedResp.Msg = errMsg
				// Use jsonx to deep merge or re-marshal
				dataBytes, _ := json.Marshal(wrappedResp)
				json.Unmarshal(dataBytes, respBody)
			}
		}
		// 服务器业务错误（code != 0）：统一包装，保持 "<action> failed: <msg>" 格式以兼容既有解析逻辑。
		return newServerCodeError(endpoint, apiErr.Code, errMsg)
	}
	// 非 2xx 且无法解析为 JSON：返回 HTTP 状态码错误（5xx/429/408 可重试）。
	return newHTTPStatusError(endpoint, resp.StatusCode, bodyStr)
}
