package middlewares

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func CustomizedLogging() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(log gin.LogFormatterParams) string {
		// your custom format
		return fmt.Sprintf("%s - [%s] \"%s %s %d %s\"\n",
			log.ClientIP,
			log.TimeStamp.Format(time.RFC1123),
			log.Method,
			log.Path,
			log.StatusCode,
			log.Latency,
		)
	})
}
