package exporter

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

type Label struct {
	Name  string
	Value string
}

type sample struct {
	labels []Label
	value  float64
}

type family struct {
	name    string
	help    string
	kind    string
	samples []sample
}

type Registry struct {
	families map[string]*family
	order    []string
}

func NewRegistry() *Registry {
	return &Registry{families: map[string]*family{}}
}

func (r *Registry) Gauge(name, help string, value float64, labels ...Label) {
	r.add(name, help, "gauge", value, labels)
}

func (r *Registry) Counter(name, help string, value float64, labels ...Label) {
	r.add(name, help, "counter", value, labels)
}

func (r *Registry) add(name, help, kind string, value float64, labels []Label) {
	f, ok := r.families[name]
	if !ok {
		f = &family{name: name, help: help, kind: kind}
		r.families[name] = f
		r.order = append(r.order, name)
	}

	f.samples = append(f.samples, sample{labels: labels, value: value})
}

func (r *Registry) WriteTo(w io.Writer) (int64, error) {
	var b strings.Builder
	names := append([]string(nil), r.order...)
	sort.Strings(names)
	for _, name := range names {
		f := r.families[name]
		_, _ = fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s %s\n", f.name, escapeHelp(f.help), f.name, f.kind)
		for _, s := range f.samples {
			_, _ = b.WriteString(f.name)
			writeLabels(&b, s.labels)
			_, _ = b.WriteString(" ")
			_, _ = b.WriteString(formatValue(s.value))
			_, _ = b.WriteString("\n")
		}
	}

	n, err := io.WriteString(w, b.String())
	return int64(n), err
}

func writeLabels(b *strings.Builder, labels []Label) {
	if len(labels) == 0 {
		return
	}

	_, _ = b.WriteString("{")
	for i, l := range labels {
		if i > 0 {
			_, _ = b.WriteString(",")
		}

		_, _ = b.WriteString(l.Name)
		_, _ = b.WriteString("=\"")
		_, _ = b.WriteString(escapeLabel(l.Value))
		_, _ = b.WriteString("\"")
	}

	_, _ = b.WriteString("}")
}

var labelEscaper = strings.NewReplacer(`\`, `\\`, "\n", `\n`, `"`, `\"`)

var helpEscaper = strings.NewReplacer(`\`, `\\`, "\n", `\n`)

func escapeLabel(s string) string { return labelEscaper.Replace(s) }

func escapeHelp(s string) string { return helpEscaper.Replace(s) }

func formatValue(v float64) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	}

	return strconv.FormatFloat(v, 'g', -1, 64)
}
