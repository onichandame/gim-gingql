package gimgingql

import (
	"io/ioutil"

	"github.com/gin-gonic/gin"
)

func bodyParser() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, _ := ioutil.ReadAll(c.Request.Body)
		c.Set("body", string(body))
		c.Next()
	}
}
