package sysctl

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const Root = "/proc/sys"

type Parameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type collector struct {
	root   string
	params []Parameter
}

func List(root string) ([]Parameter, error) {
	c := &collector{root: root}
	if err := filepath.WalkDir(root, c.visit); err != nil {
		return nil, err
	}

	sort.Slice(c.params, func(i, j int) bool { return c.params[i].Key < c.params[j].Key })

	return c.params, nil
}

func (c *collector) visit(path string, d fs.DirEntry, err error) error {
	if err != nil || d.IsDir() {
		return nil
	}

	if p, ok := read(c.root, path); ok {
		c.params = append(c.params, p)
	}

	return nil
}

func read(root, path string) (Parameter, bool) {
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm()&0o444 == 0 {
		return Parameter{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Parameter{}, false
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return Parameter{}, false
	}

	value := strings.Join(strings.Fields(string(data)), " ")

	return Parameter{Key: strings.ReplaceAll(rel, string(filepath.Separator), "."), Value: value}, true
}
