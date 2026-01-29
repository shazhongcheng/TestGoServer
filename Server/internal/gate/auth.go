package gate

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type ResumeReq struct {
	SessionId int64
	Token     string
}

func (g *Gate) signResumeToken(s *Session) string {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Sprintf("session:%d", s.ID)
	}
	nonceHex := hex.EncodeToString(nonce)
	signature := g.signTokenPayload(s.ID, nonceHex)
	return fmt.Sprintf("%s.%s", nonceHex, signature)
}

func (g *Gate) verifyToken(s *Session, token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	nonceHex := parts[0]
	signature := parts[1]
	expected := g.signTokenPayload(s.ID, nonceHex)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return false
	}
	return token == s.Token
}

func (g *Gate) secret() string { return "gate-secret" }

func (g *Gate) signTokenPayload(sessionID int64, nonceHex string) string {
	payload := fmt.Sprintf("%d:%s", sessionID, nonceHex)
	mac := hmac.New(sha256.New, []byte(g.secret()))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
