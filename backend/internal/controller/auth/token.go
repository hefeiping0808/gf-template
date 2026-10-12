package auth

import (
	"net/http"

	"backend/internal/service/auth"
	"backend/internal/service/response"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

type tokenRequest struct {
	Subject string `json:"subject"`
}

// IssueToken 是仅供本地联调的演示入口；默认关闭，正式项目应接入账号密码或 OAuth 校验。
func IssueToken(r *ghttp.Request) {
	if !g.Cfg().MustGet(r.Context(), "security.jwt.allowDemoIssue").Bool() {
		r.Response.WriteStatusExit(http.StatusForbidden, response.Error(40301, "demo token issuing is disabled"))
		return
	}
	var input tokenRequest
	if err := r.Parse(&input); err != nil || input.Subject == "" {
		r.Response.WriteStatusExit(http.StatusBadRequest, response.Error(40010, "subject is required"))
		return
	}
	token, err := auth.Issue(input.Subject)
	if err != nil {
		r.Response.WriteStatusExit(http.StatusInternalServerError, response.Error(50010, err.Error()))
		return
	}
	r.Response.WriteJson(response.OK(g.Map{"token": token, "tokenType": "Bearer"}))
}
