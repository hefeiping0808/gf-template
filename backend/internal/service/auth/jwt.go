// Package auth 提供 HTTP 与 WebSocket 共用的 JWT 签发和校验。
package auth

import (
	"errors"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
}

// Issue 为指定用户签发有时效的 HS256 JWT。
func Issue(subject string) (string, error) {
	if subject == "" {
		return "", errors.New("subject is required")
	}
	ctx := g.GetInitCtx()
	secret, err := signingKey(ctx)
	if err != nil {
		return "", err
	}
	issuer := g.Cfg().MustGet(ctx, "security.jwt.issuer").String()
	seconds := g.Cfg().MustGet(ctx, "security.jwt.expireSeconds").Int()
	if seconds <= 0 {
		seconds = 7200
	}
	now := time.Now()
	claims := Claims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer: issuer, Subject: subject,
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(seconds) * time.Second)),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// Parse 验证签名、签名算法、签发者及过期时间后返回用户标识。
func Parse(tokenString string) (string, error) {
	ctx := g.GetInitCtx()
	secret, err := signingKey(ctx)
	if err != nil {
		return "", err
	}
	issuer := g.Cfg().MustGet(ctx, "security.jwt.issuer").String()
	claims := new(Claims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected jwt signing method")
		}
		return secret, nil
	}, jwt.WithIssuer(issuer))
	if err != nil || token == nil || !token.Valid || claims.RegisteredClaims.Subject == "" {
		return "", errors.New("invalid or expired jwt")
	}
	return claims.RegisteredClaims.Subject, nil
}

func signingKey(ctx context.Context) ([]byte, error) {
	secret := []byte(g.Cfg().MustGet(ctx, "security.jwt.secret").String())
	if len(secret) < 32 || string(secret) == "REPLACE_WITH_A_RANDOM_SECRET_AT_LEAST_32_BYTES" {
		return nil, errors.New("security.jwt.secret must be replaced with a random secret of at least 32 bytes")
	}
	return secret, nil
}
