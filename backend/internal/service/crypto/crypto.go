// Package crypto 集中处理请求 SM2 解密与数据库敏感字段 AES-GCM 加密。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/emmansun/gmsm/sm2"
	"github.com/emmansun/gmsm/smx509"
	"github.com/gogf/gf/v2/frame/g"
)

// DecryptSM2 解密前端传入的 Base64 SM2 密文，默认按标准 C1C3C2 编码处理。
func DecryptSM2(ciphertext string) ([]byte, error) {
	encoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		encoded, err = base64.RawStdEncoding.DecodeString(ciphertext)
	}
	if err != nil {
		return nil, fmt.Errorf("decode sm2 ciphertext: %w", err)
	}
	keyPEM := g.Cfg().MustGet(g.GetInitCtx(), "security.sm2.privateKey").String()
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, errors.New("security.sm2.privateKey is not a valid PEM key")
	}
	parsed, err := smx509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse sm2 private key: %w", err)
	}
	privateKey, ok := parsed.(*sm2.PrivateKey)
	if !ok {
		return nil, errors.New("configured private key is not an SM2 key")
	}
	plain, err := privateKey.Decrypt(rand.Reader, encoded, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt sm2 payload: %w", err)
	}
	return plain, nil
}

// EncryptSM2 使用配置中的 SM2 公钥加密响应 data，使客户端可以与请求使用同一字段协议。
func EncryptSM2(plain []byte) (string, error) {
	keyPEM := g.Cfg().MustGet(g.GetInitCtx(), "security.sm2.publicKey").String()
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return "", errors.New("security.sm2.publicKey is not a valid PEM key")
	}
	parsed, err := smx509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse sm2 public key: %w", err)
	}
	publicKey, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("configured public key is not an SM2 key")
	}
	ciphertext, err := sm2.Encrypt(rand.Reader, publicKey, plain, nil)
	if err != nil {
		return "", fmt.Errorf("encrypt sm2 payload: %w", err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// EncryptSensitive 用 AES-256-GCM 加密数据库敏感字段，输出 aes:v1: 前缀的 Base64 密文。
func EncryptSensitive(plain []byte) (string, error) {
	aead, err := aesGCM()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, plain, nil)
	return "aes:v1:" + base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptSensitive 解密 EncryptSensitive 生成的字段，供业务读取数据库敏感数据时调用。
func DecryptSensitive(value string) ([]byte, error) {
	if !strings.HasPrefix(value, "aes:v1:") {
		return nil, errors.New("unsupported encrypted value format")
	}
	sealed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "aes:v1:"))
	if err != nil {
		return nil, err
	}
	aead, err := aesGCM()
	if err != nil {
		return nil, err
	}
	if len(sealed) < aead.NonceSize() {
		return nil, errors.New("encrypted value is too short")
	}
	nonce, ciphertext := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	return aead.Open(nil, nonce, ciphertext, nil)
}

func aesGCM() (cipher.AEAD, error) {
	configuredKey := g.Cfg().MustGet(g.GetInitCtx(), "security.aes.key").String()
	if strings.HasPrefix(configuredKey, "REPLACE_WITH_") {
		return nil, errors.New("security.aes.key placeholder must be replaced before encrypting sensitive data")
	}
	key := []byte(configuredKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("security.aes.key must be 16, 24, or 32 bytes: %w", err)
	}
	return cipher.NewGCM(block)
}
