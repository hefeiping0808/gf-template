package cmd

import (
	"context"

	"backend/internal/controller/auth"
	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	"backend/internal/controller/hello"
	"backend/internal/controller/websocket"
	"backend/internal/middleware"
	"backend/internal/service/queue"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			s := g.Server()
			if g.Cfg().MustGet(ctx, "features.rabbitmq.enabled").Bool() {
				queue.StartDefault(ctx, g.Cfg().MustGet(ctx, "features.rabbitmq.queueSize").Int(), func(jobCtx context.Context, job queue.Job) {
					g.Log().Infof(jobCtx, "local queue job processed: %s", job.Name)
				})
				defer queue.StopDefault()
			}
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse, middleware.DecryptForm, middleware.JWT)
				group.Bind(
					hello.NewV1(),
				)
				group.POST("/api/v1/auth/token", auth.IssueToken)
				group.GET("/ws", websocket.Handle)
				group.GET("/health", func(r *ghttp.Request) {
					r.Response.WriteJson(g.Map{"code": 0, "message": "ok", "data": g.Map{"status": "up"}})
				})
			})
			s.Run()
			return nil
		},
	}
)
