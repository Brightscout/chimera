package api

import (
	"encoding/json"
	"github.com/gorilla/csrf"
	"net/http"

	"github.com/pkg/errors"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	BaseURL string
	ConfirmationTemplatePath string
	CancelPagePath string
	CSRFSecret []byte
}

// RegisterAPI registers the API endpoints on the given router.
func RegisterAPI(context *Context, oauthApps map[string]OAuthApp, cache StateCache, cfg Config) (*mux.Router, error) {
	rootRouter := mux.NewRouter()

	rootRouter.Handle("/metrics", promhttp.Handler())

	rootRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ok"))
	})

	// TODO: set secure based on baseURL schema
	csrfSecure := false
	csrfHandler := csrf.Protect(cfg.CSRFSecret, csrf.Secure(csrfSecure), csrf.Path("/v1"))

	v1Router := rootRouter.PathPrefix("/v1").Subrouter()

	handler, err := NewHandler(oauthApps, cache, cfg.BaseURL, cfg.ConfirmationTemplatePath, cfg.CancelPagePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create handler")
	}

	v1Router.Handle("/auth/chimera/cancel", csrfHandler(addCtx(context.Clone(), handler.handleCancelAuthorization))).Methods(http.MethodPost)

	oauthRouter := v1Router.PathPrefix("/{provider}/{app}").Subrouter()
	oauthRouter.Handle("/oauth/authorize", addCtx(context.Clone(), handler.handleAuthorize)).Methods(http.MethodGet)
	oauthRouter.Handle("/oauth/complete", addCtx(context.Clone(), handler.handleAuthorizationCallback))
	oauthRouter.Handle("/auth/chimera/confirm", csrfHandler(addCtx(context.Clone(), handler.handleGetConfirmAuthorization))).Methods(http.MethodGet)
	oauthRouter.Handle("/auth/chimera/confirm", csrfHandler(addCtx(context.Clone(), handler.handleConfirmAuthorization))).Methods(http.MethodPost)
	oauthRouter.Handle("/oauth/token", addCtx(context.Clone(), handler.handleTokenExchange)).Methods(http.MethodPost)

	return rootRouter, nil
}

func writeJSON(w http.ResponseWriter, v interface{}, c *Context) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		c.Logger.WithError(err).Error("Failed to write json response")
	}
}
