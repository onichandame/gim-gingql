package gimgingql_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/onichandame/gim"
	gimgin "github.com/onichandame/gim-gin"
	gimgingql "github.com/onichandame/gim-gingql"
	gimgraphql "github.com/onichandame/gim-graphql"
	gqlwsclient "github.com/onichandame/gql-ws/client"
	gqlwsmessage "github.com/onichandame/gql-ws/message"
	"github.com/stretchr/testify/assert"
)

var TimerModule = gim.Module{
	Name:      "TimerModule",
	Imports:   []*gim.Module{&gimgraphql.GraphqlModule},
	Providers: []interface{}{newTimerResolver},
}

type TimerResolver struct{}

func newTimerResolver(gqlsvc *gimgraphql.GraphqlService) *TimerResolver {
	var rslv TimerResolver
	gqlsvc.AddQuery("time", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return time.Now().String(), nil
		},
	})
	gqlsvc.AddSubscription("realtime", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return p.Source, nil
		},
		Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
			c := make(chan interface{})
			go func() {
				ticker := time.NewTicker(time.Millisecond)
				for {
					select {
					case <-p.Context.Done():
						close(c)
						return
					case <-ticker.C:
						c <- time.Now().String()
					}
				}
			}()
			return c, nil
		},
	})
	return &rslv
}

func TestServer(t *testing.T) {
	root := gimgingql.NewGinGqlModule(gimgingql.Config{
		Name:    `RootModule`,
		UseWS:   true,
		Imports: []*gim.Module{&TimerModule},
	})
	root.Bootstrap()
	server := root.Get(new(gimgin.GinService)).(*gimgin.GinService).Bootstrap()
	t.Run("http handlers", func(t *testing.T) {
		t.Run("get query", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/?query={time}", nil)
			server.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			res := make(map[string]interface{})
			assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &res))
			assert.Nil(t, res["errors"])
			assert.IsType(t, "", res["data"].(map[string]interface{})["time"])
		})
		t.Run("post query", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/?query={time}", bytes.NewBufferString(`query":"{time}"`))
			server.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			res := make(map[string]interface{})
			assert.Nil(t, json.Unmarshal(w.Body.Bytes(), &res))
			assert.Nil(t, res["errors"])
			assert.IsType(t, "", res["data"].(map[string]interface{})["time"])
		})
	})
	t.Run("websocket handlers", func(t *testing.T) {
		server := httptest.NewServer(server)
		defer server.Close()
		u, err := url.Parse(server.URL)
		assert.Nil(t, err)
		u.Scheme = "ws"
		getClient := func() *gqlwsclient.Client {
			return gqlwsclient.NewClient(&gqlwsclient.Config{
				URL: u.String(),
			})
		}
		t.Run("query", func(t *testing.T) {
			client := getClient()
			defer client.Close()
			res := make(chan string)
			client.Subscribe(gqlwsmessage.SubscribePayload{Query: `query{time}`}, gqlwsclient.Handlers{OnNext: func(r *graphql.Result) { res <- r.Data.(map[string]interface{})[`time`].(string) }, OnError: func(fe gqlerrors.FormattedErrors) { close(res) }})
			v, ok := <-res
			assert.True(t, ok)
			assert.NotEmpty(t, v)
		})
		t.Run("subscription", func(t *testing.T) {
			client := getClient()
			defer client.Close()
			res := make(chan string)
			client.Subscribe(gqlwsmessage.SubscribePayload{Query: `subscription{realtime}`}, gqlwsclient.Handlers{OnNext: func(r *graphql.Result) { res <- r.Data.(map[string]interface{})[`realtime`].(string) }, OnError: func(fe gqlerrors.FormattedErrors) { close(res) }})
			for i := 0; i < 10; i++ {
				v, ok := <-res
				assert.True(t, ok)
				assert.NotEmpty(t, v)
			}
		})
	})
}
