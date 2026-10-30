package webtls

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	appctx "github.com/ChevalRouting/routier/pkg/server/api/app"
	"github.com/ChevalRouting/routier/pkg/server/tlscert"
	"github.com/ChevalRouting/routier/pkg/types"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type FileRequest struct {
	Content string `json:"content"`
}

type RenewRequest struct {
	Domain string `json:"domain"`
}

func Routes(r chi.Router) {
	r.Get("/api/web/tls", Status)
	r.Post("/api/web/tls/renew", Renew)
	r.Get("/api/web/tls/files/{name}", GetFile)
	r.Put("/api/web/tls/files/{name}", PutFile)
}

// @Summary  Web UI certificate status
// @Description Mode (self-signed, file, or acme), the fallback and ACME certificates with issuer, expiry and last error, and the files stored in /var/lib/routier/tls.
// @Tags web
// @Produce json
// @Success 200 {object} types.Response[tlscert.Status]
// @Security BearerAuth
// @Router /api/web/tls [get]
func Status(w http.ResponseWriter, r *http.Request) {
	certificates := appctx.FromContext(r.Context()).Certificates
	if certificates == nil {
		types.Err(http.StatusConflict, "the web ui is served without tls").Write(w)
		return
	}

	types.OK(w, certificates.Status())
}

// @Summary  Renew an ACME certificate now
// @Description Requests a new certificate for one of web.tls.acme.domains, even when the current one is still valid, and serves it once issued. The current certificate keeps being served if the renewal fails. Progress and errors appear in the status log.
// @Tags web
// @Produce json
// @Param body body RenewRequest true "Domain to renew"
// @Success 200 {object} types.Response[tlscert.Status]
// @Security BearerAuth
// @Router /api/web/tls/renew [post]
func Renew(w http.ResponseWriter, r *http.Request) {
	certificates := appctx.FromContext(r.Context()).Certificates
	if certificates == nil {
		types.Err(http.StatusConflict, "the web ui is served without tls").Write(w)
		return
	}

	var body RenewRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		types.Err(http.StatusBadRequest, "invalid json").Write(w)
		return
	}

	err := certificates.Renew(body.Domain)
	if errors.Is(err, tlscert.ErrNotACME) {
		types.Err(http.StatusBadRequest, err.Error()).Write(w)
		return
	}

	if errors.Is(err, tlscert.ErrRenewing) {
		types.Err(http.StatusConflict, err.Error()).Write(w)
		return
	}

	if err != nil {
		types.Err(http.StatusBadGateway, err.Error()).Write(w)
		return
	}

	types.OK(w, certificates.Status())
}

// @Summary  Read a stored web UI certificate
// @Description Returns the PEM content of a certificate stored in /var/lib/routier/tls. Files holding a private key are refused.
// @Tags web
// @Produce json
// @Param name path string true "File name"
// @Success 200 {object} types.Response[string]
// @Security BearerAuth
// @Router /api/web/tls/files/{name} [get]
func GetFile(w http.ResponseWriter, r *http.Request) {
	certificates := appctx.FromContext(r.Context()).Certificates
	if certificates == nil {
		types.Err(http.StatusConflict, "the web ui is served without tls").Write(w)
		return
	}

	content, err := certificates.ReadCertificate(chi.URLParam(r, "name"))
	if errors.Is(err, tlscert.ErrInvalidFile) {
		types.Err(http.StatusBadRequest, err.Error()).Write(w)
		return
	}

	if errors.Is(err, os.ErrNotExist) {
		types.Err(http.StatusNotFound, "file not found").Write(w)
		return
	}

	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to read tls file"))
		return
	}

	types.OK(w, content)
}

// @Summary  Store a certificate or key for the web UI
// @Description Writes a PEM file to /var/lib/routier/tls so web.tls.cert_file and key_file can name it. Private keys are stored with mode 0600.
// @Tags web
// @Produce json
// @Param name path string true "File name"
// @Param body body FileRequest true "PEM content"
// @Success 200 {object} types.Response[tlscert.Status]
// @Security BearerAuth
// @Router /api/web/tls/files/{name} [put]
func PutFile(w http.ResponseWriter, r *http.Request) {
	certificates := appctx.FromContext(r.Context()).Certificates
	if certificates == nil {
		types.Err(http.StatusConflict, "the web ui is served without tls").Write(w)
		return
	}

	var body FileRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		types.Err(http.StatusBadRequest, "invalid json").Write(w)
		return
	}

	err := certificates.StoreFile(chi.URLParam(r, "name"), []byte(body.Content))
	if errors.Is(err, tlscert.ErrInvalidFile) {
		types.Err(http.StatusBadRequest, err.Error()).Write(w)
		return
	}

	if err != nil {
		types.Error(log.Logger, w, types.Wrap(http.StatusInternalServerError, err, "failed to store tls file"))
		return
	}

	types.OK(w, certificates.Status())
}
