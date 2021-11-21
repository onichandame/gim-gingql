package gimgingql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	gimgin "github.com/onichandame/gim-gin"
	gimgraphql "github.com/onichandame/gim-graphql"
	goutils "github.com/onichandame/go-utils"
	"github.com/sirupsen/logrus"
)

type query struct {
	Query         string                 `form:"query" json:"query"`
	OperationName string                 `form:"operationName" json:"operationName"`
	Variables     map[string]interface{} `form:"variables" json:"variables"`
}

type gingqlController struct{}

func newgingqlController(logger *logrus.Entry, conf *Config, ginsvc *gimgin.GinService, gqlsvc *gimgraphql.GraphqlService) *gingqlController {
	logger = logger.WithField("scope", "GinGqlController")
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
				Context:        context.WithValue(context.Background(), reflect.TypeOf(c), c),
			})
			return res
		}))
	})
	// get handler + ws handler
	ginsvc.AddRoute(func(rg *gin.RouterGroup) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:   1024,
			WriteBufferSize:  1024,
			CheckOrigin:      func(r *http.Request) bool { return true },
			HandshakeTimeout: time.Second * 5,
		}
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
				conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
				if err != nil {
					panic(err)
				}
				defer conn.Close()
				for {
					var closed bool
					func() {
						var err error
						defer func() {
							if err != nil {
								if _, ok := err.(*websocket.CloseError); ok {
									closed = true
								} else {
									logger.Errorln(err)
									payload := make(map[string]interface{})
									errors := make([]string, 0)
									errors = append(errors, err.Error())
									payload["errors"] = errors
									conn.WriteJSON(payload)
								}
							}
						}()
						defer goutils.RecoverToErr(&err)
						msgType, msgData, err := conn.ReadMessage()
						if err != nil {
							panic(err)
						}
						if msgType != websocket.TextMessage {
							panic(fmt.Errorf("message type %v not supported. required types: %v", msgType, websocket.TextMessage))
						}
						var q query
						if err := json.Unmarshal(msgData, &q); err != nil {
							panic(err)
						}
						isSubscription := strings.HasPrefix(strings.TrimSpace(q.Query), `subscription`)
						if isSubscription {
							reschan := graphql.Subscribe(graphql.Params{
								Schema:         *schema,
								RequestString:  q.Query,
								VariableValues: q.Variables,
								OperationName:  q.OperationName,
								Context:        context.WithValue(context.Background(), reflect.TypeOf(c), c),
							})
							go func() {
								for res := range reschan {
									if conn.WriteJSON(res) != nil {
										break
									}
								}
							}()
						} else {
							res := graphql.Do(graphql.Params{
								Schema:         *schema,
								RequestString:  q.Query,
								VariableValues: q.Variables,
								OperationName:  q.OperationName,
								Context:        context.WithValue(context.Background(), reflect.TypeOf(c), c),
							})
							conn.WriteJSON(res)
						}
					}()
					if closed {
						fmt.Println("hi")
						break
					}
				}
			} else {
				var q query
				goutils.Assert(c.Bind(&q))
				res := graphql.Do(graphql.Params{
					Schema:        *schema,
					RequestString: string(q.Query),
					OperationName: q.OperationName,
					Context:       context.WithValue(context.Background(), reflect.TypeOf(c), c),
				})
				c.JSON(200, res)
			}
		})
	})
	return &ctl
}
