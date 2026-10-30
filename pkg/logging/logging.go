package logging

import (
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

func Console() zerolog.Logger {
	writer := zerolog.ConsoleWriter{
		Out:                 os.Stderr,
		TimeFormat:          "15:04:05",
		FormatErrFieldValue: indentErrorLines,
	}

	return zerolog.New(writer).With().Timestamp().Logger()
}

func indentErrorLines(v any) string {
	s, _ := v.(string)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		if unquoted, err := strconv.Unquote(s); err == nil {
			s = unquoted
		}
	}

	lines := strings.Split(s, "\n")
	if len(lines) <= 1 {
		return s
	}

	return "\n  " + strings.Join(lines, "\n  ")
}
