package reroute

import (
	"context"
	"net/http"
	"path"
	"strings"
)

const (
	varsKey = iota
)

func badRequest(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 " + http.StatusText(400)))
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write([]byte("405 " + http.StatusText(405)))
}

func internalServerError(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(500)
	w.Write([]byte("500 " + http.StatusText(500)))
}

func Vars(r *http.Request) map[string]string {
	vars := r.Context().Value(varsKey)
	if vars != nil {
		return vars.(map[string]string)
	}
	return nil
}

func (r *router) PreparePath(p string) string {
	if r.cleanPath {
		if p == "" {
			p = "/"
		}
		if p[0] != '/' {
			p = "/" + p
		}
		p = path.Clean(p)

		if !r.trimTrailingSlashes && p[len(p)-1] == '/' && p != "/" {
			p += "/"
		}
		return p
	}

	if r.trimTrailingSlashes {
		p = strings.TrimRight(p, "/")
	}
	return p
}

type router struct {
	routes              []*route
	trimTrailingSlashes bool
	cleanPath           bool

	BadRequestHandler          http.Handler
	NotFoundHandler            http.Handler
	MethodNotAllowedHandler    http.Handler
	InternalServerErrorHandler http.Handler
}

func NewRouter() *router {
	router := new(router)

	router.trimTrailingSlashes = true
	router.cleanPath = true

	router.BadRequestHandler = http.HandlerFunc(badRequest)
	router.MethodNotAllowedHandler = http.HandlerFunc(methodNotAllowed)
	router.InternalServerErrorHandler = http.HandlerFunc(internalServerError)
	router.NotFoundHandler = http.NotFoundHandler()

	return router
}

func (r *router) CleanPath(clean bool) *router {
	r.cleanPath = clean
	return r
}

func (r *router) TrimTrailingSlashes(trim bool) *router {
	r.trimTrailingSlashes = trim
	return r
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var matchedRoute *route
	wrongMethod := false
	for _, route := range r.routes {
		match, method := route.match(req)
		if match && !method {
			matchedRoute = route
			wrongMethod = false
			break
		} else if match && method {
			wrongMethod = method
		}
	}

	if wrongMethod {
		r.MethodNotAllowedHandler.ServeHTTP(w, req)
		return
	}

	if matchedRoute == nil || matchedRoute.Handler == nil {
		r.NotFoundHandler.ServeHTTP(w, req)
		return
	}

	req = req.WithContext(context.WithValue(req.Context(), varsKey, matchedRoute.Vars))

	if matchedRoute.Filters.Run(w, req) {
		matchedRoute.Handler.ServeHTTP(w, req)
	}
}

func (r *router) Handle(regex string, handler http.Handler) *route {
	if regex[len(regex) - 1] != '$' {
		regex += "$"
	}

	return r.NewRoute(regex, handler)
}

func (r *router) HandleFunc(
	regex string,
	f func(w http.ResponseWriter, req *http.Request),
) *route {
	return r.Handle(regex, http.HandlerFunc(f))
}

func (r *router) HandlePartial(regex string, handler http.Handler) *route {
	return r.NewRoute(regex, handler)
}

func (r *router) HandleFuncPartial(
	regex string,
	f func(w http.ResponseWriter, req *http.Request),
) *route {
	return r.HandlePartial(regex, http.HandlerFunc(f))
}

func (r *router) NewRoute(regexp string, handler http.Handler) *route {
	if regexp[0] != '^' {
		regexp = "^" + regexp
	}

	route := &route{
		r,
		regexp,
		handler,
		make(map[string]string),
		new(FilterChain),
		nil,
	}
	r.routes = append(r.routes, route)
	return route
}
