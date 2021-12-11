package gimgingql

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	gimgin "github.com/onichandame/gim-gin"
	gimgraphql "github.com/onichandame/gim-graphql"
	goutils "github.com/onichandame/go-utils"
	gqlwsserver "github.com/onichandame/gql-ws/server"
	"github.com/sirupsen/logrus"
)

type query struct {
	Query         string                 `form:"query" json:"query"`
	OperationName string                 `form:"operationName" json:"operationName"`
	Variables     map[string]interface{} `form:"variables" json:"variables"`
}

type gingqlController struct{}

func newgingqlController(logger *logrus.Entry, conf *Config, ginsvc *gimgin.GinService, gqlsvc *gimgraphql.GraphqlService) *gingqlController {
	var ctl gingqlController
	schema := gqlsvc.BuildSchema()
	// basic post handler
	ginsvc.AddRoute(func(rg *gin.RouterGroup) {
		rg.Use(bodyParser())
		rg.POST(conf.Endpoint, gimgin.GetHTTPHandler(func(c *gin.Context) interface{} {
			var q query
			goutils.Assert(json.Unmarshal([]byte(c.GetString("body")), &q))
			res := graphql.Do(graphql.Params{
				Schema:         *schema,
				RequestString:  q.Query,
				OperationName:  q.OperationName,
				VariableValues: q.Variables,
				Context:        context.WithValue(context.Background(), ginCtxKey, c),
			})
			return res
		}))
	})
	// get handler + ws handler
	ginsvc.AddRoute(func(rg *gin.RouterGroup) {
		rg.GET(conf.Endpoint, func(c *gin.Context) {
			if c.IsWebsocket() {
				var err error
				defer func() {
					if err != nil {
						c.Error(err)
						err = nil
					}
				}()
				defer goutils.RecoverToErr(&err)
				if !conf.UseWS {
					panic(fmt.Errorf("websocket handler not enabled"))
				}
				sock := gqlwsserver.NewSocket(&gqlwsserver.Config{
					Response: c.Writer, 
					Request: c.Request, 
					Schema: schema, Context: context.WithValue(context.Background(), ginCtxKey, c),
				})
				sock.Wait()
			} else {
				var q query
				goutils.Assert(c.Bind(&q))
				res := graphql.Do(graphql.Params{
					Schema:        *schema,
					RequestString: string(q.Query),
					OperationName: q.OperationName,
					Context:       context.WithValue(context.Background(), ginCtxKey, c),
				})
				c.JSON(200, res)
			}
		})
	})
	return &ctl
}
