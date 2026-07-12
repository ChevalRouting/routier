package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func renderServiceConfigs(data TemplateData, funcs template.FuncMap) ([]Output, error) {
	var out []Output
	for svcName, svc := range data.Services {
		if !svc.Enable {
			continue
		}

		for _, sc := range svc.Configs {
			content, err := renderServiceTemplate(sc.Src, data, funcs)
			if err != nil {
				return nil, fmt.Errorf("services.%s template %s: %w", svcName, sc.Src, err)
			}

			out = append(out, Output{
				Name:    "service:" + svcName + "/" + filepath.Base(sc.Src),
				Dest:    sc.Dest,
				Content: content,
			})
		}
	}

	return out, nil
}

func renderServiceTemplate(src string, data TemplateData, funcs template.FuncMap) (string, error) {
	clean := filepath.Clean(src)
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("template path %q is invalid", src)
	}

	userPath := filepath.Join(UserDir, clean)
	var raw []byte
	var err error

	if _, e := os.Stat(userPath); e == nil {
		raw, err = os.ReadFile(userPath)
	} else {
		raw, err = embedded.ReadFile("defaults/" + clean)
	}

	if err != nil {
		return "", fmt.Errorf("template %s: %w", src, err)
	}

	t, err := template.New(src).Funcs(funcs).Parse(string(raw))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
