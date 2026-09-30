package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/api/requests"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

const apiKeyTokenPrefix = "rtr_api_"

func randomAPIKey() (string, string, error) {
	idBytes := make([]byte, 16)
	secret := make([]byte, 32)
	if _, err := rand.Read(idBytes); err != nil {
		return "", "", err
	}

	if _, err := rand.Read(secret); err != nil {
		return "", "", err
	}

	return hex.EncodeToString(idBytes), apiKeyTokenPrefix + base64.RawURLEncoding.EncodeToString(secret), nil
}

// ListAPIKeys godoc
// @Summary List application API keys
// @Tags auth
// @Produce json
// @Success 200 {object} types.Response[[]types.APIKey]
// @Security BearerAuth
// @Router /api/auth/api-keys [get]
func ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	keys, err := webdb.ListAPIKeys(r.Context(), app.DB)
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "list API keys"))
		return
	}

	types.OK(w, keys)
}

// CreateAPIKey godoc
// @Summary Create an application API key
// @Tags auth
// @Produce json
// @Param body body types.CreateAPIKeyRequest true "API key name"
// @Success 200 {object} types.Response[types.CreateAPIKeyResponse]
// @Security BearerAuth
// @Router /api/auth/api-keys [post]
func CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	var request types.CreateAPIKeyRequest
	if validationError := app.DecodeAndValidate(r, &request); validationError != nil {
		validationError.Write(w)
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		types.Err(http.StatusBadRequest, "name is required").Write(w)
		return
	}

	id, token, err := randomAPIKey()
	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "generate API key"))
		return
	}

	key := types.APIKey{
		ID: id, Name: request.Name, Prefix: token[:16], CreatedAt: time.Now().Unix(),
		CreatedBy: appctx.UsernameFromContext(r.Context()),
	}
	if err := webdb.CreateAPIKey(requests.DurableContext(r), app.DB, key, token); err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "create API key"))
		return
	}

	types.OK(w, types.CreateAPIKeyResponse{APIKey: key, Token: token})
}

// RevokeAPIKey godoc
// @Summary Revoke an application API key
// @Tags auth
// @Produce json
// @Param id path string true "API key id"
// @Success 200 {object} types.Response[types.StatusResponse]
// @Security BearerAuth
// @Router /api/auth/api-keys/{id} [delete]
func RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	app := appctx.FromContext(r.Context())
	err := webdb.RevokeAPIKey(requests.DurableContext(r), app.DB, chi.URLParam(r, "id"))
	if errors.Is(err, webdb.ErrAPIKeyNotFound) {
		types.Err(http.StatusNotFound, "API key not found").Write(w)
		return
	}

	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "revoke API key"))
		return
	}

	types.OK(w, types.StatusResponse{Status: "ok"})
}
