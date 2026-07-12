package svc

import (
	"fmt"
	"strings"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/rs/zerolog/log"
)

func EnableServices(services map[string]*config.Service, dryRun bool) error {
	for name, s := range services {
		if !s.Enable {
			continue
		}

		if err := enableService(name, dryRun); err != nil {
			return err
		}
	}

	return nil
}

func ReloadServices(services map[string]*config.Service, changedNames []string, dryRun bool) error {
	for name, s := range services {
		if !s.Enable {
			continue
		}

		hasChange := false
		prefix := "service:" + name + "/"
		for _, n := range changedNames {
			if strings.HasPrefix(n, prefix) {
				hasChange = true
				break
			}
		}

		if !ServiceRunning(name) || hasChange {
			if err := restartService(name, dryRun); err != nil {
				return err
			}
		}
	}

	return nil
}

func enableService(name string, dryRun bool) error {
	args := []string{"rc-update", "add", name, "default"}
	if dryRun {
		log.Info().Strs("cmd", args).Msg("would run")
		return nil
	}

	log.Info().Str("service", name).Msg("rc-update add")
	_ = runCmd(args, 0)
	return nil
}

func restartService(name string, dryRun bool) error {
	verb := "restart"
	if getServiceStatus(name) == svcStopped {
		verb = "start"
	}

	args := []string{"rc-service", name, verb}
	if dryRun {
		log.Info().Strs("cmd", args).Msg("would run")
		return nil
	}

	log.Info().Str("service", name).Msg(verb)
	if err := runCmd(args, 0); err != nil {
		return fmt.Errorf("%s %s: %w", verb, name, err)
	}

	return nil
}
