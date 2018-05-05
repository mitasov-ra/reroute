package reroute

import (
	"log"
	"net/http"
	"regexp"
	"fmt"
)

type route struct {
	Router     *router
	PathRegexp string
	Handler    http.Handler
	Vars       map[string]string
	Filters    *FilterChain
	methods    []string
}

func (r *route) match(req *http.Request) (bool, bool) {
	exp := regexp.MustCompile(r.PathRegexp) //prevent running with bad routes

	path := r.Router.PreparePath(req.URL.Path)

	if !exp.MatchString(path) {
		return false, false
	} else if !r.matchMethods(req) {
		return true, true
	}

	r.ExtractVarsWithRegex(exp, path)

	return true, false
}

func (r *route) ExtractVars(regExpr string, string string) {
	exp := regexp.MustCompile(regExpr)

	//collecting Vars from regexp
	match := exp.FindStringSubmatch(string)

	for i, name := range exp.SubexpNames() {
		if i != 0 && name != "" {
			r.Vars[name] = match[i]
		}
	}
}

func (r *route) ExtractVarsWithRegex(exp *regexp.Regexp, string string) {
	//collecting Vars from regexp
	match := exp.FindStringSubmatch(string)

	for i, name := range exp.SubexpNames() {
		if i != 0 && name != "" {
			r.Vars[name] = match[i]
		}
	}
}

func (r *route) matchMethods(req *http.Request) bool {
	if r.methods == nil {
		return true
	}

	for _, m := range r.methods {
		if req.Method == m {
			return true
		}
	}

	return false
}

func (r *route) Methods(methods ...string) *route {
	r.methods = methods
	return r
}

func (r *route) Schemes(scheme string) *route {
	r.Filters.AddFunc(func(w http.ResponseWriter, req *http.Request, chain *FilterChain) bool {
		if req.URL.Scheme == scheme {
			return chain.Run(w, req)
		}

		r.Router.NotFoundHandler.ServeHTTP(w, req)
		return false
	})

	return r
}

func (r *route) Host(schemeRegex string) *route {
	exp := regexp.MustCompile(schemeRegex)

	r.Filters.AddFunc(func(w http.ResponseWriter, req *http.Request, chain *FilterChain) bool {
		host := req.URL.Host

		if exp.MatchString(host) {
			r.ExtractVarsWithRegex(exp, host)
			return chain.Run(w, req)
		}

		r.Router.NotFoundHandler.ServeHTTP(w, req)
		return false
	})

	return r
}

func (r *route) Headers(pairs ...string) *route {
	headersMap, err := mapFromPairs(pairs...)
	if err != nil {
		panic(err.Error() + " at Headers")
	}

	r.Filters.AddFunc(func(writer http.ResponseWriter, request *http.Request, chain *FilterChain) bool {
		for k, v := range headersMap {
			if v != request.Header.Get(k) {
				r.Router.BadRequestHandler.ServeHTTP(writer, request)
				return false
			}
		}

		return chain.Run(writer, request)
	})

	return r
}

func (r *route) Queries(pairs ...string) *route {
	queryMap, err := mapFromPairs(pairs...)
	if err != nil {
		panic(err.Error() + " at Queries")
	}

	r.Filters.AddFunc(func(writer http.ResponseWriter, request *http.Request, chain *FilterChain) bool {
		for k, v := range queryMap {
			if v != "" {
				v = "(?:" + v + ")(?:$|&)"
			}

			if k == "" {
				r.Router.InternalServerErrorHandler.ServeHTTP(writer, request)
				log.Fatal("Error 500: Empty parameter name in Queries filter")
				return false
			} else {
				k = "(?:^|&)" + k
			}

			pattern := k + "=" + v

			if match, err := regexp.MatchString(pattern, request.URL.RawQuery);
				match && err == nil {
				continue
			} else if err != nil {
				r.Router.InternalServerErrorHandler.ServeHTTP(writer, request)
				log.Fatal(err)
				return false
			} else {
				r.Router.BadRequestHandler.ServeHTTP(writer, request)
				return false
			}
		}

		return chain.Run(writer, request)
	})

	return r
}

func mapFromPairs(pairs ...string) (map[string]string, error) {
	ln := len(pairs)
	if (ln & 1) != 0 {
		return nil, fmt.Errorf(
			"reroute: number of parameters must be multiple of 2, got %v",
			ln,
		)
	}
	resMap := map[string]string{}
	for i := 0; i < ln; i += 2 {
		resMap[pairs[i]] = pairs[i+1]
	}

	return resMap, nil
}
