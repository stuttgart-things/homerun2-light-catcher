package handlers

import "net/http"

// Mount serves app under basePath (e.g. "/light"), the way homerun2-portal
// forwards it: the request path arrives unchanged, with the prefix. The health
// endpoints stay at the root for the probes and the portal's status check, and
// / leads to the dashboard. An empty basePath serves app at the root as before.
func Mount(basePath string, app, health http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/health", health)
	mux.Handle("/healthz", health)
	if basePath == "" {
		mux.Handle("/", app)
		return mux
	}
	toUI := http.RedirectHandler(basePath+"/", http.StatusTemporaryRedirect)
	mux.Handle(basePath+"/", http.StripPrefix(basePath, app))
	mux.Handle(basePath, toUI)
	mux.Handle("/{$}", toUI)
	return mux
}
