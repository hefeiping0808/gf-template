// Package websocket 提供需要握手 JWT 与逐消息 SM2 加密的 WebSocket 示例服务。
package websocket

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"backend/internal/service/auth"
	cryptoservice "backend/internal/service/crypto"
	"backend/internal/service/response"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024, WriteBufferSize: 1024,
	CheckOrigin: func(request *http.Request) bool {
		origin := request.Header.Get("Origin")
		if origin == "" {
			return true
		}
		parsed, err := url.Parse(origin)
		return err == nil && parsed.Host == request.Host
	},
}

type message struct {
	Token string `json:"token"`
	Event string `json:"event"`
	Data  string `json:"data"`
}

// Handle 建立连接后逐条校验 JWT；鉴权失败或 data 解密失败时返回统一格式错误并关闭连接。
func Handle(r *ghttp.Request) {
	handshakeToken := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if handshakeToken == "" {
		handshakeToken = r.Get("token").String()
	}
	subject, err := auth.Parse(handshakeToken)
	if err != nil {
		r.Response.WriteStatusExit(http.StatusUnauthorized, response.Error(40101, "unauthorized"))
		return
	}
	conn, err := upgrader.Upgrade(r.Response, r.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	readLimit := g.Cfg().MustGet(r.Context(), "websocket.readLimitBytes").Int64()
	if readLimit <= 0 {
		readLimit = 1 << 20
	}
	conn.SetReadLimit(readLimit)
	for {
		_, raw, readErr := conn.ReadMessage()
		if readErr != nil {
			return
		}
		var input message
		if json.Unmarshal(raw, &input) != nil || input.Token == "" || input.Event == "" || input.Data == "" {
			writeError(conn, 40020, "message must contain token, event and encrypted data")
			return
		}
		messageSubject, parseErr := auth.Parse(input.Token)
		if parseErr != nil || messageSubject != subject {
			writeError(conn, 40102, "invalid message token")
			return
		}
		plain, decryptErr := cryptoservice.DecryptSM2(input.Data)
		if decryptErr != nil || !json.Valid(plain) {
			writeError(conn, 40021, "unable to decrypt message data")
			return
		}
		payload, _ := json.Marshal(map[string]any{"event": input.Event, "data": json.RawMessage(plain)})
		ciphertext, encryptErr := cryptoservice.EncryptSM2(payload)
		if encryptErr != nil {
			writeError(conn, 50020, "unable to encrypt response data")
			return
		}
		if err = conn.WriteJSON(response.OK(ciphertext)); err != nil {
			return
		}
	}
}

func writeError(conn *websocket.Conn, code int, message string) {
	_ = conn.WriteJSON(response.Error(code, message))
}
