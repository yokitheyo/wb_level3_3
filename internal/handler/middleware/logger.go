package middleware

import (
	"time"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"
)

func LoggerMiddleware() ginext.HandlerFunc {
	return func(c *ginext.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		zlog.Logger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Msg("HTTP request")
	}
}
