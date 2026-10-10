package middlewares

import (
	"fmt"
	"runtime/debug"

	"github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func HandlePanic(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(string(debug.Stack()))
			logrus.Error(err)
			c.Error(errors.ErrInternalServer)
			c.Abort()
		}
	}()
	c.Next()
}
