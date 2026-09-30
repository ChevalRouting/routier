package managers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/ChevalRouting/routier/pkg/apply"
	"github.com/ChevalRouting/routier/pkg/applylog"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/failures"
	"github.com/ChevalRouting/routier/pkg/render"
	"github.com/ChevalRouting/routier/pkg/svc"
	"github.com/ChevalRouting/routier/pkg/types"
	anyk "github.com/m-vinc/anyk"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var applyMu sync.Mutex

const LastAppliedPath = "/var/lib/routier/last-applied.yml"

func saveLastApplied(cfg *config.Config) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return
	}

	if err := os.WriteFile(LastAppliedPath, data, 0600); err != nil {
		log.Warn().Err(err).Msg("save last-applied config")
	}
}

const applyLockFile = "/var/lib/routier/apply.lock"

type ApplyOptions struct {
	DryRun        bool
	ReloadAll     bool
	SkipWireguard bool
	Source        string
	ConfigPath    string
}

func acquireApplyLock(ctx context.Context) (func(), error) {
	applyMu.Lock()

	if err := os.MkdirAll("/var/lib/routier", 0700); err != nil {
		applyMu.Unlock()
		return nil, fmt.Errorf("create routier dir: %w", err)
	}

	f, err := os.OpenFile(applyLockFile, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		applyMu.Unlock()
		return nil, fmt.Errorf("open apply lock: %w", err)
	}

	for {
		if ferr := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); ferr == nil {
			break
		} else if ferr != syscall.EWOULDBLOCK {
			f.Close()
			applyMu.Unlock()
			return nil, fmt.Errorf("acquire apply lock: %w", ferr)
		}

		select {
		case <-ctx.Done():
			f.Close()
			applyMu.Unlock()
			return nil, fmt.Errorf("waiting for apply lock: %w", ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}

	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		applyMu.Unlock()
	}, nil
}

func rawArtifactErrors(err error) []types.ArtifactError {
	var ve *failures.ValidationError
	if errors.As(err, &ve) {
		return ve.Errors
	}

	return []types.ArtifactError{{Message: err.Error()}}
}

func ApplyConfig(ctx context.Context, cfg *config.Config, outputs []render.Output, opts ApplyOptions) (snapID string, changed []string, err error) {
	source := opts.Source
	if source == "" {
		source = "cli"
	}

	var rec *applylog.Recorder
	if !opts.DryRun {
		release, lerr := acquireApplyLock(ctx)
		if lerr != nil {
			return "", nil, lerr
		}

		defer release()

		rec = applylog.Start(source, opts.ConfigPath)
		defer func() {
			result := "applied"
			if err != nil {
				result = "failed"
				if snapID != "" {
					_ = failures.Save(rec.ID(), source, snapID, cfg, outputs, rawArtifactErrors(err))
					rec.MarkBundle()
					log.Warn().Str("bundle", rec.ID()).Msg("apply failed, saved rendered artifacts")
				}
			}

			rec.Finish(snapID, result)
		}()
	}

	if !opts.DryRun {
		if verrs := svc.ValidateArtifactsBeforeApply(outputs); len(verrs) > 0 {
			_ = failures.Save(rec.ID(), source, "", cfg, outputs, verrs)
			rec.MarkBundle()
			ve := failures.NewValidationError(verrs)
			ve.BundleID = rec.ID()
			return "", nil, ve
		}
	}

	if err := apply.Hostname(cfg.Hostname, opts.DryRun); err != nil {
		return "", nil, err
	}

	if len(cfg.Users) > 0 {
		if err := apply.Users(cfg.Users, opts.DryRun); err != nil {
			return "", nil, err
		}
	}

	if err := apply.DNS(cfg.DNS, cfg.Hostname, opts.DryRun); err != nil {
		return "", nil, err
	}

	if len(outputs) > 0 {
		snapshotAlso := []string{LastAppliedPath}
		configPath := opts.ConfigPath
		if configPath == "" {
			configPath = "/etc/routier/config.yml"
		}

		if info, serr := os.Stat(configPath); serr == nil && info.Mode().IsRegular() {
			snapshotAlso = append(snapshotAlso, configPath)
		}

		snapID, changed, err = apply.Write(outputs, opts.DryRun, snapshotAlso...)
		if err != nil {
			return "", nil, err
		}
	}

	reloadNames := changed
	if opts.ReloadAll {
		reloadNames = make([]string, 0, len(outputs))
		for _, o := range outputs {
			reloadNames = append(reloadNames, o.Name)
		}
	}

	if err := svc.ReloadFromOutputs(reloadNames, cfg, opts.DryRun, opts.SkipWireguard); err != nil {
		return snapID, changed, err
	}

	if err := svc.ReconcileServices(cfg, opts.DryRun); err != nil {
		return snapID, changed, err
	}

	if len(cfg.Services) > 0 {
		if err := svc.EnableServices(cfg.Services, opts.DryRun); err != nil {
			return snapID, changed, err
		}

		if err := svc.ReloadServices(cfg.Services, reloadNames, opts.DryRun); err != nil {
			return snapID, changed, err
		}
	}

	if !opts.DryRun {
		saveLastApplied(cfg)
	}

	return snapID, changed, nil
}

func toAnykServices(services []config.AnycastService) []anyk.AnykService {
	out := make([]anyk.AnykService, len(services))
	for i, svc := range services {
		eps := make([]anyk.AnykEndpoint, len(svc.Endpoints))
		for j, ep := range svc.Endpoints {
			ae := anyk.AnykEndpoint{IP: ep.IP, Distance: ep.Distance}
			if ep.HTTPCheck != nil {
				ae.HTTPCheck = &anyk.AnykHTTPCheck{
					Verb:         ep.HTTPCheck.Verb,
					URL:          ep.HTTPCheck.URL,
					ExpectedCode: ep.HTTPCheck.ExpectedCode,
					Headers:      ep.HTTPCheck.Headers,
					Body:         ep.HTTPCheck.Body,
					Timeout:      ep.HTTPCheck.Timeout,
				}
			}

			if ep.DNSCheck != nil {
				ae.DNSCheck = &anyk.AnykDNSCheck{
					Resolver: ep.DNSCheck.Resolver,
					Type:     ep.DNSCheck.Type,
					Query:    ep.DNSCheck.Query,
					Expected: ep.DNSCheck.Expected,
					Timeout:  ep.DNSCheck.Timeout,
				}
			}

			eps[j] = ae
		}

		out[i] = anyk.AnykService{
			Name:       svc.Name,
			Active:     svc.Active,
			AnycastIPs: svc.AnycastIPs,
			Endpoints:  eps,
		}
	}

	return out
}
