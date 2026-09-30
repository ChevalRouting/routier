package config

import (
	"net/http"

	"github.com/ChevalRouting/routier/pkg/config/layers"
	"github.com/ChevalRouting/routier/pkg/config/layers/registry"
	"github.com/ChevalRouting/routier/pkg/types"
)

func requestLayer(r *http.Request) (layers.Layer, *types.AppError) {
	name := r.URL.Query().Get("layer")
	if name == "" {
		name = "advanced"
	}

	layer, ok := registry.Get(name)
	if !ok {
		return nil, types.Errorf(http.StatusBadRequest, "unknown configuration layer: %s", name)
	}

	return layer, nil
}
