package gate

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

const tokenTTL = 5 * time.Minute

type ResumeReq struct {
	SessionId int64
	Token     string
}

func (g *Gate) signResumeToken(s *Session) string {
	payload := fmt.Sprintf("%d:%d", s.ID, time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(g.secret()))
	mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString([]byte(payload + ":" + fmt.Sprintf("%x", mac.Sum(nil))))
}

func (g *Gate) verifyToken(s *Session, token string) bool {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return false
	}
	parts := string(raw)
	var sessionID int64
	var ts int64
	var sig string
	if _, err := fmt.Sscanf(parts, "%d:%d:%s", &sessionID, &ts, &sig); err != nil {
		return false
	}
	if sessionID != s.ID {
		return false
	}
	if time.Since(time.Unix(ts, 0)) > tokenTTL {
		return false
	}
	payload := fmt.Sprintf("%d:%d", sessionID, ts)
	mac := hmac.New(sha256.New, []byte(g.secret()))
	mac.Write([]byte(payload))
	expected := fmt.Sprintf("%x", mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

func (g *Gate) secret() string {
	return "gate-secret"
}
