package diffutil

import (
	"sort"
	"strings"

	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func Lines(before, after string) []types.DiffLine {
	dmp := diffmatchpatch.New()
	ca, cb, lineMap := dmp.DiffLinesToChars(before, after)
	diffs := dmp.DiffCharsToLines(dmp.DiffMain(ca, cb, false), lineMap)

	out := []types.DiffLine{}
	for _, d := range diffs {
		typ := "same"

		switch d.Type {
		case diffmatchpatch.DiffInsert:
			typ = "add"
		case diffmatchpatch.DiffDelete:
			typ = "remove"
		}

		for _, line := range strings.Split(strings.TrimRight(d.Text, "\n"), "\n") {
			out = append(out, types.DiffLine{Type: typ, Text: line})
		}
	}

	return out
}

func Files(before, after map[string]string) []types.FileDiff {
	names := map[string]bool{}
	for n := range before {
		names[n] = true
	}

	for n := range after {
		names[n] = true
	}

	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}

	sort.Strings(sorted)

	out := []types.FileDiff{}
	for _, name := range sorted {
		b, a := before[name], after[name]
		if b == a {
			continue
		}

		status := "modified"
		switch {
		case b == "":
			status = "added"
		case a == "":
			status = "removed"
		}

		out = append(out, types.FileDiff{File: name, Status: status, Lines: Lines(b, a)})
	}

	return out
}
