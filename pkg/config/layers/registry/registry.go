package registry

import (
	"github.com/ChevalRouting/routier/pkg/config/layers"
	"github.com/ChevalRouting/routier/pkg/config/layers/advanced"
	"github.com/ChevalRouting/routier/pkg/config/layers/simple"
)

var registered = map[string]layers.Layer{
	"advanced": advanced.Layer{},
	"simple":   simple.Layer{},
}

func Get(name string) (layers.Layer, bool) {
	layer, ok := registered[name]
	return layer, ok
}

func Names() []string {
	return []string{"advanced", "simple"}
}
