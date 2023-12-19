package main

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func Test_Routes_Exists(t *testing.T) {
	testApp := Config{}

	testAppRoutes := testApp.routes()
	chiRoutes := testAppRoutes.(chi.Router)

	if chiRoutes == nil {
		t.Error("routes not found")
	}

	routes := []string{"/authenticate"}

	testRoutes(t, routes, chiRoutes)

}

func testRoutes(t *testing.T, routes []string, router chi.Router) {
	for _, route := range routes {
		_ = chi.Walk(router, func(method string, foundRoute string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {

			if route != foundRoute {
				t.Errorf("route %s not found", route)
			}
			return nil
		})
	}
}
