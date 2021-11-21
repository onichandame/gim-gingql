package gimgingql

import "github.com/onichandame/gim"

type Config struct {
	// when true subscription is enabled
	UseWS    bool
	Name     string
	Endpoint string
	Imports  []*gim.Module
}
