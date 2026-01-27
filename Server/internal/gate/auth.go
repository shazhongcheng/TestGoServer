package gate

import "fmt"

type ResumeReq struct {
	SessionId int64
	Token     string
}

func (g *Gate) signResumeToken(s *Session) string {
	return fmt.Sprintf("session:%d", s.ID)
}

func (g *Gate) verifyToken(s *Session, token string) bool {
	return token == s.Token
}

func (g *Gate) secret() string { return "gate-secret" }
