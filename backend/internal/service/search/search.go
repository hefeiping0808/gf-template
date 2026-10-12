// Package search 提供不依赖 Elasticsearch 的轻量数据库全文匹配入口。
package search

import (
	"context"
	"errors"
	"regexp"
	"strconv"

	"github.com/gogf/gf/v2/frame/g"
)

var identifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// InTable 在给定表和字段上执行参数化 LIKE 搜索；表名和字段名先做白名单格式校验。
// 复杂全文检索可后续替换为 Redis 索引或专用搜索服务。
func InTable(ctx context.Context, table, field, keyword string, limit int) ([]map[string]any, error) {
	if !g.Cfg().MustGet(ctx, "features.search.enabled").Bool() {
		return nil, errors.New("search feature is disabled")
	}
	if g.Cfg().MustGet(ctx, "features.search.driver").String() != "mysql" {
		return nil, errors.New("search driver is not mysql; configure a redis index adapter for redis mode")
	}
	if !identifier.MatchString(table) || !identifier.MatchString(field) {
		return nil, errors.New("invalid table or field identifier")
	}
	if keyword == "" {
		return nil, errors.New("keyword is required")
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	sql := "SELECT * FROM `" + table + "` WHERE `" + field + "` LIKE ? LIMIT " + gconvInt(limit)
	rows, err := g.DB().GetAll(ctx, sql, "%"+keyword+"%")
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.Map())
	}
	return result, nil
}

func gconvInt(value int) string {
	return strconv.Itoa(value)
}
