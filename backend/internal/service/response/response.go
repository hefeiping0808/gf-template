// Package response 定义 REST 与 WebSocket 共用的响应结构。
package response

// Envelope 是所有 JSON 接口统一使用的返回外壳。
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 返回成功响应；新增接口应复用该结构保持前端解析方式一致。
func OK(data any) Envelope {
	return Envelope{Code: 0, Message: "ok", Data: data}
}

// Error 返回业务错误响应。
func Error(code int, message string) Envelope {
	return Envelope{Code: code, Message: message, Data: nil}
}
