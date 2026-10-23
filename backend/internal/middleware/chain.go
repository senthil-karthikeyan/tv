// Package middleware provides composable HTTP middleware functions for the server.
package middleware

import "net/http"

// Chain wraps handler h in the provided middlewares.
// The first middleware in the list becomes the outermost wrapper, so the call order is:
//
//	middlewares[0] → middlewares[1] → … → h
func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	// Apply in reverse so that middlewares[0] ends up outermost.
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
