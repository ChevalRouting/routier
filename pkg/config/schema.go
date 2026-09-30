package config

import (
	"reflect"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
)

var (
	schemaOnce sync.Once
	schema     *jsonschema.Schema
	schemaErr  error
)

func Schema() (*jsonschema.Schema, error) {
	schemaOnce.Do(func() {
		schema, schemaErr = jsonschema.ForType(reflect.TypeOf(Config{}), &jsonschema.ForOptions{IgnoreInvalidTypes: true})
	})

	return schema, schemaErr
}
