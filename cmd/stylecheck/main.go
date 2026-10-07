package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

type options struct {
	root  string
	only  string
	paths []string
}

func main() {
	root := flag.String("root", ".", "path inside the repository")
	only := flag.String("only", "all", "check group: all, go, style, text, web")
	flag.Parse()
	if !slices.Contains([]string{"all", "go", "style", "text", "web"}, *only) {
		_, _ = fmt.Fprintln(os.Stderr, "unknown check group:", *only)
		os.Exit(2)
	}

	if len(flag.Args()) > 0 && *only != "style" && *only != "text" {
		_, _ = fmt.Fprintln(os.Stderr, "file arguments require --only=style or --only=text")
		os.Exit(2)
	}

	command := exec.Command("git", "-C", *root, "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "locate repository:", err)
		os.Exit(2)
	}

	settings := options{root: strings.TrimSpace(string(output)), only: *only, paths: flag.Args()}
	os.Exit(runChecks(settings))
}

func runChecks(settings options) int {
	failed := false
	if settings.only == "all" || settings.only == "go" {
		if !runCommand(settings.root, ".", "golangci-lint", "config", "verify") {
			failed = true
		} else {
			failed = !runCommand(settings.root, ".", "golangci-lint", "run", "--config", ".golangci.yml", "./...") || failed
		}
	}

	if settings.only == "all" || settings.only == "style" || settings.only == "text" {
		paths, err := sourcePaths(settings)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			failed = true
		} else {
			failed = !checkFiles(settings, paths) || failed
		}
	}

	if settings.only == "all" || settings.only == "web" {
		failed = !runCommand(settings.root, "web", "npm", "run", "lint") || failed
	}

	if failed {
		return 1
	}

	return 0
}

func runCommand(root, directory string, arguments ...string) bool {
	_, _ = fmt.Printf("%s: %s\n", directory, strings.Join(arguments, " "))
	command := exec.Command(arguments[0], arguments[1:]...)
	command.Dir = filepath.Join(root, directory)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return false
	}

	return true
}

func sourcePaths(settings options) ([]string, error) {
	if len(settings.paths) > 0 {
		return settings.paths, nil
	}

	command := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	command.Dir = settings.root
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("list repository files: %w", err)
	}

	paths := strings.Split(strings.TrimSuffix(string(output), "\x00"), "\x00")
	slices.Sort(paths)
	return slices.Compact(paths), nil
}

func checkFiles(settings options, paths []string) bool {
	failed := false
	for _, path := range paths {
		if path == "" || generatedPath(path) || strings.HasSuffix(path, ".tsbuildinfo") {
			continue
		}

		absolute := path
		if !filepath.IsAbs(path) {
			absolute = filepath.Join(settings.root, path)
		}

		data, err := os.ReadFile(absolute)
		if os.IsNotExist(err) && len(settings.paths) == 0 {
			continue
		}

		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			failed = true
			continue
		}

		var issues []diagnostic
		if utf8.Valid(data) && !strings.ContainsRune(string(data), '\x00') {
			issues = append(issues, checkText(path, data)...)
		}

		if settings.only != "text" && strings.HasSuffix(path, ".go") {
			issues = append(issues, checkSource(path, data)...)
		}

		for _, issue := range issues {
			_, _ = fmt.Fprintf(os.Stderr, "%s:%d: %s: %s\n", issue.path, issue.line, issue.rule, issue.message)
			failed = true
		}
	}

	return !failed
}

func generatedPath(path string) bool {
	normalized := "/" + filepath.ToSlash(filepath.Clean(path))
	return strings.Contains(normalized, "/vendor/") || strings.HasSuffix(normalized, "/vendor") || strings.Contains(normalized, "/node_modules/") || strings.Contains(normalized, "/dist/") || strings.Contains(normalized, "/pkg/db/generated/") || strings.Contains(normalized, "/web/src/api/") || strings.Contains(normalized, "/pkg/server/client/generated/") || strings.Contains(normalized, "/pkg/server/api/docs/") || strings.HasPrefix(normalized, "/migrated/")
}

func checkText(path string, data []byte) []diagnostic {
	var issues []diagnostic
	for index, line := range strings.Split(string(data), "\n") {
		if strings.ContainsRune(line, '\u2014') {
			issues = append(issues, diagnostic{path: path, line: index + 1, rule: "text", message: "replace the em dash with a comma, colon or parentheses"})
		}
	}

	return issues
}
