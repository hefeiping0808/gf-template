// Package storage 将对象存储需求映射为本地目录文件存储。
package storage

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

// Save 将文件写入配置目录并返回相对存储键；key 不允许逃逸出存储根目录。
func Save(key string, source io.Reader) error {
	if !g.Cfg().MustGet(g.GetInitCtx(), "features.cos.enabled").Bool() {
		return errors.New("local storage feature is disabled")
	}
	root := g.Cfg().MustGet(g.GetInitCtx(), "features.cos.localDir").String()
	if root == "" {
		return errors.New("features.cos.localDir is required")
	}
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("invalid storage key")
	}
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	target := filepath.Join(rootPath, clean)
	if err = os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, source)
	return err
}

// Open 打开已存储文件，key 路径校验与 Save 保持一致。
func Open(key string) (*os.File, error) {
	if !g.Cfg().MustGet(g.GetInitCtx(), "features.cos.enabled").Bool() {
		return nil, errors.New("local storage feature is disabled")
	}
	root := g.Cfg().MustGet(g.GetInitCtx(), "features.cos.localDir").String()
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, errors.New("invalid storage key")
	}
	return os.Open(filepath.Join(root, clean))
}
