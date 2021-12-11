package gimgingql

import (
	"context"

	"github.com/gin-gonic/gin"
)

var ginCtxKey = &struct{}{}

func GetGinCtx(c context.Context) *gin.Context {
	return c.Value(ginCtxKey).(*gin.Context)
}
