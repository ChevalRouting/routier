package cfgstore

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ChevalRouting/routier/pkg/api/friendcache"
	"github.com/ChevalRouting/routier/pkg/config"
	"github.com/ChevalRouting/routier/pkg/managers"
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

func StagingLayerPath(configPath, username string) string {
	return StagingPath(configPath, username) + ".layer"
}

type LayerConflictError struct {
	Owner     string
	Requested string
}

func (e *LayerConflictError) Error() string {
	return fmt.Sprintf("staged configuration belongs to %s layer, requested %s", e.Owner, e.Requested)
}

func StagingError(err error, message string) *types.AppError {
	var conflict *LayerConflictError
	if errors.As(err, &conflict) {
		return types.Errorf(http.StatusConflict, "configuration changes belong to the %s layer", conflict.Owner)
	}

	return types.Wrap(http.StatusInternalServerError, err, message)
}

func StagingLayer(configPath, username string) (string, bool, error) {
	Mu.RLock()
	defer Mu.RUnlock()

	return stagingLayer(configPath, username)
}

func stagingLayer(configPath, username string) (string, bool, error) {
	if _, err := os.Stat(StagingPath(configPath, username)); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}

		return "", false, err
	}

	data, err := os.ReadFile(StagingLayerPath(configPath, username))
	if err != nil {
		if os.IsNotExist(err) {
			return "advanced", true, nil
		}

		return "", false, err
	}

	layer := strings.TrimSpace(string(data))
	if layer == "" {
		layer = "advanced"
	}

	return layer, true, nil
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

func ReadLayer(configPath, username, layer string) (*config.Config, error) {
	Mu.RLock()
	defer Mu.RUnlock()

	owner, exists, err := stagingLayer(configPath, username)
	if err != nil {
		return nil, err
	}

	path := configPath
	if exists && owner == layer {
		path = StagingPath(configPath, username)
	}

	return config.Load(path)
}

func WriteStaging(configPath, username string, cfg *config.Config) error {
	return WriteStagingLayer(configPath, username, "advanced", cfg)
}

func WriteStagingLayer(configPath, username, layer string, cfg *config.Config) error {
	Mu.Lock()
	defer Mu.Unlock()

	owner, exists, err := stagingLayer(configPath, username)
	if err != nil {
		return err
	}
	if exists && owner != layer {
		return &LayerConflictError{Owner: owner, Requested: layer}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if current, err := os.ReadFile(configPath); err == nil && string(current) == string(data) {
		_ = os.Remove(StagingPath(configPath, username))
		_ = os.Remove(StagingLayerPath(configPath, username))
		return nil
	}

	path := StagingPath(configPath, username)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}

	if err := os.WriteFile(StagingLayerPath(configPath, username), []byte(layer+"\n"), 0600); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func WriteLive(configPath string, cfg *config.Config) error {
	if errs := config.Validate(cfg, false); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}

		return fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
	}

	Mu.Lock()
	defer Mu.Unlock()

	return config.Save(configPath, cfg)
}

func Promote(configPath, username string) error {
	Mu.Lock()
	defer Mu.Unlock()

	stagingPath := StagingPath(configPath, username)
	data, err := os.ReadFile(stagingPath)
	if err != nil {
		return fmt.Errorf("read staging: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	return os.WriteFile(configPath, data, 0600)
}

func PromoteConfig(configPath string, cfg *config.Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	Mu.Lock()
	defer Mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(configPath), 0750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	return os.WriteFile(configPath, data, 0600)
}

func Discard(configPath, username string) {
	Mu.Lock()
	defer Mu.Unlock()
	_ = os.Remove(StagingPath(configPath, username))
	_ = os.Remove(StagingLayerPath(configPath, username))
}

func DiscardLayer(configPath, username, layer string) error {
	Mu.Lock()
	defer Mu.Unlock()

	owner, exists, err := stagingLayer(configPath, username)
	if err != nil {
		return err
	}
	if exists && owner != layer {
		return &LayerConflictError{Owner: owner, Requested: layer}
	}

	if err := os.Remove(StagingPath(configPath, username)); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(StagingLayerPath(configPath, username)); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func Resolve(cfg *config.Config) (*config.Config, error) {
	return managers.Resolve(cfg, friendcache.InterpolationVars())
}
