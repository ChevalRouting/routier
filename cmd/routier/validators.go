package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
)

type validationFunc func() error

func validateConfig(fns ...validationFunc) error {
	for _, fn := range fns {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

func validateLogLevel() error {
	if _, err := zerolog.ParseLevel(cli.LogLevel); err != nil {
		return fmt.Errorf("invalid log level %q (use debug, info, warn, error)", cli.LogLevel)
	}

	return nil
}

func validateConfigFileFunc(path string) validationFunc {
	return func() error { return validateConfigFileFuncCallback(path) }
}

func validateConfigFileFuncCallback(path string) error {
	s, err := os.Stat(path)
	if err != nil || s.IsDir() {
		return fmt.Errorf("unable to open config file %q: %w", path, err)
	}

	return nil
}
