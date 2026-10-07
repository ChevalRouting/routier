package config

import (
	"net/http"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/rs/zerolog/log"
)

type schemaDocument struct {
	Version string             `json:"version"`
	Schema  *jsonschema.Schema `json:"schema"`
}

type schemaVersionDocument struct {
	Version string `json:"version"`
}

// @Summary  Get the JSON Schema for the canonical configuration
// @Tags config
// @Produce json
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/config/schema [get]
func GetSchema(w http.ResponseWriter, _ *http.Request) {
	schema, err := cfgpkg.Schema()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to build config schema"))
		return
	}

	types.OK(w, schemaDocument{Version: cfgpkg.CurrentVersion, Schema: schema})
}

// @Summary  Get the version of the canonical configuration schema
// @Tags config
// @Produce json
// @Success 200 {object} object
// @Security BearerAuth
// @Router /api/config/schema/version [get]
func GetSchemaVersion(w http.ResponseWriter, _ *http.Request) {
	types.OK(w, schemaVersionDocument{Version: cfgpkg.CurrentVersion})
}
