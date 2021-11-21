package gimgingql

import (
	"context"
	"reflect"

	"github.com/gin-gonic/gin"
)

func GetGinCtx(c context.Context) *gin.Context {
	return c.Value(reflect.TypeOf(new(gin.Context))).(*gin.Context)
}
