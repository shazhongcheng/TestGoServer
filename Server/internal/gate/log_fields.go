package gate

import "go.uber.org/zap"

func sessionFields(s *Session) []zap.Field {
	if s == nil {
		return []zap.Field{
			zap.Int64("session", 0),
			zap.Int64("player", 0),
		}
	}
	return []zap.Field{
		zap.Int64("session", s.ID),
		zap.Int64("player", s.PlayerID),
	}
}

func connFields(c *Conn) []zap.Field {
	if c == nil {
		return []zap.Field{
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		}
	}
	return []zap.Field{
		zap.Int64("conn_id", c.id),
		zap.String("trace_id", c.traceID),
	}
}
