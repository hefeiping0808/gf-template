// Package middleware 实现统一请求加密解密与 JWT 认证。
package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"backend/internal/service/auth"
	cryptoservice "backend/internal/service/crypto"
	"backend/internal/service/response"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

const defaultMaxRequestBody = 8 << 20

// DecryptForm 解密写请求顶层 data 字段；token 申请接口保留明文以便用户建立登录态。
func DecryptForm(r *ghttp.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
		r.Middleware.Next()
		return
	}
	if r.URL.Path == "/api/v1/auth/token" || strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		r.Middleware.Next()
		return
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	if mediaType == "application/json" {
		decryptJSON(r)
		return
	}
	if mediaType == "application/x-www-form-urlencoded" {
		decryptURLEncoded(r)
		return
	}
	if mediaType == "multipart/form-data" {
		decryptMultipart(r)
		return
	}
	r.Middleware.Next()
}

func decryptJSON(r *ghttp.Request) {
	maxBodyBytes := maxRequestBodyBytes(r)
	body, err := io.ReadAll(io.LimitReader(r.Request.Body, maxBodyBytes+1))
	if err != nil || int64(len(body)) > maxBodyBytes {
		reject(r, http.StatusBadRequest, 40001, "invalid request body")
		return
	}
	var envelope map[string]json.RawMessage
	if err = json.Unmarshal(body, &envelope); err != nil || envelope == nil {
		reject(r, http.StatusBadRequest, 40002, "request body must be a JSON object with encrypted data")
		return
	}
	var encrypted string
	if err = json.Unmarshal(envelope["data"], &encrypted); err != nil || encrypted == "" {
		reject(r, http.StatusBadRequest, 40003, "encrypted data field is required")
		return
	}
	plain, err := decryptData(encrypted)
	if err != nil {
		reject(r, http.StatusBadRequest, 40004, "unable to decrypt request data")
		return
	}
	envelope["data"] = json.RawMessage(plain)
	decoded, err := json.Marshal(envelope)
	if err != nil {
		reject(r, http.StatusBadRequest, 40005, "invalid request data")
		return
	}
	r.Request.Body = io.NopCloser(bytes.NewReader(decoded))
	r.Request.ContentLength = int64(len(decoded))
	r.Middleware.Next()
}

func decryptURLEncoded(r *ghttp.Request) {
	maxBodyBytes := maxRequestBodyBytes(r)
	body, err := io.ReadAll(io.LimitReader(r.Request.Body, maxBodyBytes+1))
	if err != nil || int64(len(body)) > maxBodyBytes {
		reject(r, http.StatusBadRequest, 40001, "invalid request body")
		return
	}
	values, err := url.ParseQuery(string(body))
	if err != nil || values.Get("data") == "" {
		reject(r, http.StatusBadRequest, 40003, "encrypted data field is required")
		return
	}
	plain, err := decryptData(values.Get("data"))
	if err != nil {
		reject(r, http.StatusBadRequest, 40004, "unable to decrypt request data")
		return
	}
	values.Set("data", string(plain))
	decoded := []byte(values.Encode())
	r.Request.Body = io.NopCloser(bytes.NewReader(decoded))
	r.Request.ContentLength = int64(len(decoded))
	r.Request.PostForm, r.Request.Form = values, values
	r.Middleware.Next()
}

func decryptMultipart(r *ghttp.Request) {
	maxBodyBytes := maxRequestBodyBytes(r)
	r.Request.Body = http.MaxBytesReader(r.Response, r.Request.Body, maxBodyBytes)
	if err := r.Request.ParseMultipartForm(maxBodyBytes); err != nil {
		reject(r, http.StatusBadRequest, 40001, "invalid multipart form")
		return
	}
	defer r.Request.MultipartForm.RemoveAll()
	values := r.Request.MultipartForm.Value
	if len(values["data"]) == 0 || values["data"][0] == "" {
		reject(r, http.StatusBadRequest, 40003, "encrypted data field is required")
		return
	}
	plain, err := decryptData(values["data"][0])
	if err != nil {
		reject(r, http.StatusBadRequest, 40004, "unable to decrypt request data")
		return
	}
	values["data"][0] = string(plain)
	r.Request.PostForm, r.Request.Form = r.Request.MultipartForm.Value, r.Request.MultipartForm.Value
	r.Middleware.Next()
}

func maxRequestBodyBytes(r *ghttp.Request) int64 {
	// 更新时间：2026-10-13 01:47:37 CST；限制加密表单体积，避免上传内容耗尽服务内存。
	maxBytes := int64(0)
	if configured, err := g.Cfg().Get(r.Context(), "request.maxBodyBytes"); err == nil {
		maxBytes = configured.Int64()
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxRequestBody
	}
	return maxBytes
}

func decryptData(encrypted string) ([]byte, error) {
	plain, err := cryptoservice.DecryptSM2(encrypted)
	if err != nil || !json.Valid(plain) {
		return nil, errInvalidEncryptedData
	}
	return plain, nil
}

func reject(r *ghttp.Request, status, code int, message string) {
	r.Response.WriteStatusExit(status, response.Error(code, message))
}

var errInvalidEncryptedData = errors.New("decrypted data is not valid json")

// JWT 校验 HTTP Bearer token；WebSocket 握手允许 query token，消息仍需各自携带 token。
func JWT(r *ghttp.Request) {
	path := r.URL.Path
	if path == "/health" || path == "/hello" || path == "/api/v1/auth/token" {
		r.Middleware.Next()
		return
	}
	tokenString := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if path == "/ws" && tokenString == "" {
		tokenString = r.Get("token").String()
	}
	subject, err := auth.Parse(tokenString)
	if err != nil {
		r.Response.WriteStatusExit(http.StatusUnauthorized, response.Error(40101, "unauthorized"))
		return
	}
	r.SetCtxVar("auth.subject", subject)
	r.Middleware.Next()
}
