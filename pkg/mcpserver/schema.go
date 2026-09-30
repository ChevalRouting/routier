package mcpserver

import (
	"context"
	"fmt"
	"net/http"

	cfgpkg "github.com/ChevalRouting/routier/pkg/config"
)

type schemaDocument struct {
	Version string `json:"version"`
	Schema  any    `json:"schema"`
}

func embeddedSchema() (schemaDocument, error) {
	schema, err := cfgpkg.Schema()
	if err != nil {
		return schemaDocument{}, err
	}

	return schemaDocument{Version: cfgpkg.CurrentVersion, Schema: schema}, nil
}

func (i *instance) schemaVersion(ctx context.Context) (string, error) {
	result, err := i.request(ctx, http.MethodGet, "/api/config/schema/version", "", nil)
	if err != nil {
		return "", err
	}

	m, ok := result.(map[string]any)
	if !ok {
		return "", fmt.Errorf("%s: unexpected schema version response", i.name)
	}

	version, _ := m["version"].(string)
	return version, nil
}
