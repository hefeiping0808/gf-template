package cmd

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"os"

	"backend/internal/service/user"

	"github.com/gogf/gf/v2/os/gcmd"
	"golang.org/x/term"
)

func init() {
	if err := Main.AddCommand(createAdminCommand()); err != nil {
		panic(err)
	}
}

func createAdminCommand() *gcmd.Command {
	return &gcmd.Command{
		Name:        "create-admin",
		Usage:       "main create-admin --username <name>",
		Brief:       "创建管理员账号",
		Description: "自动创建 users 表，并安全读取密码后创建 bcrypt 哈希管理员账号。",
		Arguments: []gcmd.Argument{{
			Name:  "username",
			Short: "u",
			Brief: "管理员用户名",
		}},
		Strict: true,
		Func:   runCreateAdmin,
	}
}

func runCreateAdmin(ctx context.Context, parser *gcmd.Parser) error {
	usernameOption := parser.GetOpt("username")
	if usernameOption == nil || usernameOption.String() == "" {
		return errors.New("必须指定 --username，例如：go run . create-admin --username admin")
	}
	username := usernameOption.String()
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("create-admin 需要交互式终端以安全读取密码")
	}
	password, err := readHiddenPassword("管理员密码： ")
	if err != nil {
		return err
	}
	defer clear(password)
	confirmation, err := readHiddenPassword("确认管理员密码： ")
	if err != nil {
		return err
	}
	defer clear(confirmation)
	if subtle.ConstantTimeCompare(password, confirmation) != 1 {
		return errors.New("两次输入的密码不一致")
	}
	if err = user.CreateAdmin(ctx, username, password); err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "管理员 %q 创建成功。\n", username)
	return err
}

func readHiddenPassword(prompt string) ([]byte, error) {
	if _, err := fmt.Fprint(os.Stderr, prompt); err != nil {
		return nil, err
	}
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(os.Stderr)
	return password, err
}
