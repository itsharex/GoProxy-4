package webapi

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const jwtHeader = `{"alg":"HS256","typ":"JWT"}`

type jwtClaims struct {
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}

type tokenIssuer struct {
	mu       sync.RWMutex
	secret   []byte
	expire   time.Duration
	validateFn func(username, password string) (passwordHash string, err error)
}

func newTokenIssuer(validateFn func(username, password string) (string, error), secret string, expireHours int) *tokenIssuer {
	ti := &tokenIssuer{
		expire:     time.Duration(expireHours) * time.Hour,
		validateFn: validateFn,
	}
	if secret != "" {
		ti.secret = []byte(secret)
	} else {
		ti.secret = generateSecret()
	}
	return ti
}

func generateSecret() []byte {
	b := make([]byte, 32)
	rand.Read(b)
	return b
}

func (ti *tokenIssuer) Authenticate(username, password string) (string, error) {
	hash, err := ti.validateFn(username, password)
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", errors.New("用户名或密码错误")
	}
	return ti.issue(username)
}

func (ti *tokenIssuer) issue(username string) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		Username: username,
		Exp:      now.Add(ti.expire).Unix(),
		Iat:      now.Unix(),
	}
	headerEnc := base64urlEncode([]byte(jwtHeader))
	claimsData, _ := json.Marshal(claims)
	claimsEnc := base64urlEncode(claimsData)
	signingInput := headerEnc + "." + claimsEnc
	sig := ti.sign([]byte(signingInput))
	return signingInput + "." + base64urlEncode(sig), nil
}

func (ti *tokenIssuer) Validate(tokenStr string) (*jwtClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("token 格式无效")
	}
	signingInput := parts[0] + "." + parts[1]
	sig, err := base64urlDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("token 签名解码失败: %w", err)
	}
	if !ti.verify([]byte(signingInput), sig) {
		return nil, errors.New("token 签名无效")
	}
	claimsData, err := base64urlDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("token 载荷解码失败: %w", err)
	}
	var claims jwtClaims
	if err := json.Unmarshal(claimsData, &claims); err != nil {
		return nil, fmt.Errorf("token 载荷解析失败: %w", err)
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token 已过期")
	}
	return &claims, nil
}

func (ti *tokenIssuer) SetSecret(secret []byte) {
	ti.mu.Lock()
	ti.secret = secret
	ti.mu.Unlock()
}

func (ti *tokenIssuer) sign(data []byte) []byte {
	h := hmac.New(sha256.New, ti.secret)
	h.Write(data)
	return h.Sum(nil)
}

func (ti *tokenIssuer) verify(data, sig []byte) bool {
	expected := ti.sign(data)
	return hmac.Equal(sig, expected)
}

func base64urlEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64urlDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
