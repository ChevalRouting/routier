package cfgstore

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
	"github.com/ChevalRouting/routier/pkg/server/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/types"
	"gopkg.in/yaml.v3"
)

func ValidationError(cfg *config.Config) *types.AppError {
	resolved, err := Resolve(cfg)
	if err != nil {
		return types.Errorf(http.StatusBadRequest, "config interpolation: %v", err)
	}

	errs := config.Validate(resolved, false)
	if len(errs) == 0 {
		return nil
	}

	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}

	return types.NewError(http.StatusBadRequest, strings.Join(msgs, "; "))
}

var Mu sync.RWMutex

func StagingPath(configPath, username string) string {
	return configPath + ".pending." + username
}

func StagingError(err error, message string) *types.AppError {
	return types.Wrap(http.StatusInternalServerError, err, message)
}

func ApplyError(err error, code int, message string) *types.AppError {
	if errors.Is(err, managers.ErrApplyPending) {
		return types.NewError(http.StatusConflict, managers.ErrApplyPending.Error())
	}

	return types.Wrap(code, err, message)
}

func HasStaging(configPath, username string) (bool, error) {
	Mu.RLock()
	defer Mu.RUnlock()

	_, err := os.Stat(StagingPath(configPath, username))
	if os.IsNotExist(err) {
		return false, nil
	}

	return err == nil, err
}

func Read(configPath, username string) (*config.Config, error) {
	Mu.RLock()
	defer Mu.RUnlock()

	path := StagingPath(configPath, username)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = configPath
	}

	return config.Load(path)
}

func ReadRunning(configPath string) (*config.Config, error) {
	Mu.RLock()
	defer Mu.RUnlock()

	return config.Load(configPath)
}

func WriteStaging(configPath, username string, cfg *config.Config) error {
	Mu.Lock()
	defer Mu.Unlock()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if current, err := os.ReadFile(configPath); err == nil && string(current) == string(data) {
		_ = os.Remove(StagingPath(configPath, username))
		return nil
	}

	path := StagingPath(configPath, username)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func validateLive(cfg *config.Config) error {
	errs := config.Validate(cfg, false)
	if len(errs) == 0 {
		return nil
	}

	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}

	return fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
}

func MarshalLive(cfg *config.Config) ([]byte, error) {
	if err := validateLive(cfg); err != nil {
		return nil, err
	}

	return yaml.Marshal(cfg)
}

func WriteLive(configPath string, cfg *config.Config) error {
	if err := validateLive(cfg); err != nil {
		return err
	}

	if err := managers.RequireNoPending(); err != nil {
		return err
	}

	Mu.Lock()
	defer Mu.Unlock()

	return config.Save(configPath, cfg)
}

func Discard(configPath, username string) {
	Mu.Lock()
	defer Mu.Unlock()
	_ = os.Remove(StagingPath(configPath, username))
}

func DiscardApplied(configPath, username string) error {
	Mu.Lock()
	defer Mu.Unlock()

	path := StagingPath(configPath, username)
	staged, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	committed, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	if !bytes.Equal(staged, committed) {
		return nil
	}

	return os.Remove(path)
}

func Resolve(cfg *config.Config) (*config.Config, error) {
	return managers.Resolve(cfg, friendcache.InterpolationVars())
}
