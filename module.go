package gimgingql

import (
	"github.com/onichandame/gim"
	gimgin "github.com/onichandame/gim-gin"
	gimgraphql "github.com/onichandame/gim-graphql"
)

func NewGinGqlModule(conf Config) *gim.Module {
	var mod gim.Module
	mod.Name = conf.Name
	mod.Imports = []*gim.Module{&gimgin.GinModule, &gimgraphql.GraphqlModule}
	mod.Imports = append(mod.Imports, conf.Imports...)
	mod.Providers = []interface{}{newgingqlController, &conf, newLogger}
	return &mod
}
