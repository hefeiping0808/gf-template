package hello

import (
	"context"

	"backend/api/hello/v1"
)

func (c *ControllerV1) Hello(ctx context.Context, req *v1.HelloReq) (res *v1.HelloRes, err error) {
	// 更新时间：2026-10-13 01:40:00 CST；返回数据交给统一响应中间件包装。
	return &v1.HelloRes{Message: "Hello World!"}, nil
}
