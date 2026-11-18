package config

import (
	"encoding/json"
	"reflect"
	"strings"
)

var secretFields = map[string]bool{
	"auth_key":      true,
	"eab_key":       true,
	"key":           true,
	"password":      true,
	"password_hash": true,
	"preshared_key": true,
	"private_key":   true,
	"secret":        true,
	"token":         true,
}

var redactExempt = reflect.TypeFor[VRRPInstance]()

func Redacted(cfg *Config) (*Config, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	var out Config
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}

	out.BaseDir = cfg.BaseDir
	redactValue(reflect.ValueOf(&out).Elem())

	return &out, nil
}

func redactValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			redactValue(v.Elem())
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			redactValue(v.Index(i))
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			elem := v.MapIndex(k)
			if elem.Kind() == reflect.Pointer {
				redactValue(elem)
			}
		}
	case reflect.Struct:
		redactStruct(v)
	}
}

func redactStruct(v reflect.Value) {
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if secretFields[name] && field.Type.Kind() == reflect.String && t != redactExempt {
			v.Field(i).SetString("")
			continue
		}

		redactValue(v.Field(i))
	}
}
