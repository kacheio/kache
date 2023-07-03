package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kacheio/kache/pkg/utils/version"
)

// VersionHandler exposes version routes.
type VersionHandler struct{}

// Append adds version routes to the specified router.
func (v VersionHandler) Append(router *mux.Router) {
	router.Methods(http.MethodGet).PathPrefix("/version").HandlerFunc(version.Handler)
}
