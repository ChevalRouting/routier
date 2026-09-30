package layers

import (
	"encoding/json"

	"github.com/ChevalRouting/routier/pkg/config"
)

type Projection struct {
	Layer    string         `json:"layer"`
	Sections map[string]any `json:"sections"`
	Issues   []Issue        `json:"issues,omitempty"`
	Losses   []Loss         `json:"losses,omitempty"`
}

type Issue struct {
	Section string   `json:"section"`
	Paths   []string `json:"paths,omitempty"`
	Message string   `json:"message"`
}

type Loss struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type Layer interface {
	Name() string
	Sections() []string
	Project(*config.Config) (*Projection, error)
	ProjectSection(*config.Config, string) (any, error)
	Build(json.RawMessage) (*config.Config, error)
	ReplaceSection(*config.Config, string, json.RawMessage) (*config.Config, error)
}
