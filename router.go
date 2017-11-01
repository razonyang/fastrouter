// Copyright 2017 Razon Yang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fastrouter

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Trailing slashes policies.
const (
	// ignore trailing slashes.
	IgnoreTrailingSlashes = iota

	// append trailing slashes and redirect if request path is not end with '/'.
	AppendTrailingSlashes

	// remove trailing slashes and redirect if request path is end with '/'.
	RemoveTrailingSlashes

	// remove or append trailing slashes according to corresponding pattern.
	StrictTrailingSlashes
)

// ParamsKey is an empty struct, it is the second parameter of
// context.WithValue for storing the request parameters.
type ParamsKey struct{}

var contextParamsKey ParamsKey

// New returns a new Router with the default parser
// via NewWithParser.
func New() *Router {
	return NewWithParser(NewParser())
}

// NewWithParser returns a new Router with the given
// parser.
func NewWithParser(parser ParserInterface) *Router {
	return &Router{
		combinedRegexps:       make(map[string]*regexp.Regexp),
		groups:                make(map[string]*Router),
		parser:                parser,
		routes:                make(map[string][]*route),
		TrailingSlashesPolicy: IgnoreTrailingSlashes,
	}
}

// Router is an implementation of http.Handler for handling HTTP requests.
type Router struct {
	// parent router.
	parent *Router

	// Middleware.
	Middleware []Middleware

	// mapping from request method to combined regular expression.
	combinedRegexps map[string]*regexp.Regexp

	// mapping from prefix to group router.
	groups map[string]*Router

	// mapping from request method to []route.
	routes map[string][]*route

	// pattern parser.
	parser ParserInterface

	// The handler for handling panic.
	//
	// The rcv contains panic information, rcv = recover().
	//
	// This options is only effective in root router.
	PanicHandler func(w http.ResponseWriter, req *http.Request, rcv interface{})

	// The handler for handling OPTIONS request.
	//
	// The methods contains all allowed methods of the request path.
	//
	// This options is only effective in root router.
	OptionsHandler func(w http.ResponseWriter, req *http.Request, methods []string)

	// The handler for handling Method Not Allowed.
	//
	// The methods contains all allowed methods of the request path.
	//
	// This options is only effective in root router.
	MethodNotAllowedHandler func(w http.ResponseWriter, req *http.Request, methods []string)

	// The handler for handling Not Found.
	//
	// This options is only effective in root router.
	NotFoundHandler http.Handler

	// Trailing slashes policy:
	//     IgnoreTrailingSlashes, by default
	//     AppendTrailingSlashes
	//     RemoveTrailingSlashes
	//     StrictTrailingSlashes
	//
	// This options is only effective in root router.
	TrailingSlashesPolicy int8
}

// Prepare makes preparations before handling requests:
//
// 1. combines route's regular expressions;
//
// 2. chaining middleware.
//
// Note that, router MUST makes preparations before handling request,
// otherwise it can not works as expected.
func (r *Router) Prepare() {
	r.prepare()
}

func (r *Router) prepare() {
	// retrieve middleware for chaining
	middleware := r.middleware()

	for method := range r.routes {
		routes := r.routes[method]
		regs := []string{}
		for i := 0; i < len(routes); i++ {
			if routes[i] != nil {
				regs = append(regs, "("+routes[i].reg+")")

				// chaining middleware
				handler := routes[i].handler
				// handler middleware
				for j := len(routes[i].middleware) - 1; j >= 0; j-- {
					handler = routes[i].middleware[j](handler)
				}
				// global middleware
				for j := len(middleware) - 1; j >= 0; j-- {
					handler = middleware[j](handler)
				}
				routes[i].finalHandler = handler
				routes[i].finalHandler = handler
			}
		}
		reg := strings.Join(regs, "|")
		r.combinedRegexps[method] = regexp.MustCompile("^(?:" + reg + ")$")
	}

	for _, group := range r.groups {
		group.prepare()
	}
}

// Group returns a new group router with then given prefix.
func (r *Router) Group(prefix string) *Router {
	if prefix == "" {
		panic(`the group prefix MUST NOT be empty`)
	}
	if strings.Contains(prefix, "/") {
		panic(`the group prefix MUST NOT contains '/'`)
	}

	if _, ok := r.groups[prefix]; ok {
		panic(fmt.Errorf("the group which prefix equal to %q already exists", prefix))
	}

	// group will inherits parent's parser
	group := New()
	group.parent = r
	group.parser = r.parser
	r.groups[prefix] = group
	return group
}

// Handle registers handler with the given method, pattern and middleware.
//
// The request method is case sensitive.
//
// The handler is a http.HandlerFunc that handle request.
//
// It also allows to specify middleware for the given handler, for example,
// we usually specify a body limit middleware for the upload handler.
//
// Causes a panic if parsing failed, such as invalid pattern.
func (r *Router) Handle(method, pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	if _, ok := r.routes[method]; !ok {
		r.routes[method] = []*route{nil}
	}
	route := &route{handler: handler, middleware: middleware}
	var err error
	route.reg, route.params, route.hasTrailingSlashes, err = r.parser.Parse(pattern)
	if err != nil {
		panic(err)
	}

	r.routes[method] = append(r.routes[method], route)
	for i := 0; i < len(route.params); i++ {
		r.routes[method] = append(r.routes[method], nil)
	}
}

// Delete is a shortcut of Handle for handling DELETE request.
func (r *Router) Delete(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	r.Handle(http.MethodDelete, pattern, handler, middleware...)
}

// Get is a shortcut of Handle for handling GET request.
func (r *Router) Get(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	r.Handle(http.MethodGet, pattern, handler, middleware...)
}

// Post is a shortcut of Handle for handling POST request.
func (r *Router) Post(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	r.Handle(http.MethodPost, pattern, handler, middleware...)
}

// Put is a shortcut of Handle for handling PUT request.
func (r *Router) Put(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	r.Handle(http.MethodPut, pattern, handler, middleware...)
}

// ServeFiles serve static resources.
//
// The pattern MUST contains parameter placeholder named "filepath",
// it is related to pattern parser.
//
// The root is the absolute or relative path of the static resources.
func (r *Router) ServeFiles(pattern, root string, middleware ...Middleware) {
	if !strings.Contains(pattern, "filepath") {
		panic(`the pattern MUST contains parameter placeholder named "filepath"`)
	}

	fs := http.FileServer(http.Dir(root))
	handler := func(w http.ResponseWriter, req *http.Request) {
		if params, ok := req.Context().Value(contextParamsKey).(map[string]string); ok {
			req.URL.Path = params["filepath"]
			fs.ServeHTTP(w, req)
			return
		}
	}

	r.Handle(http.MethodGet, pattern, http.HandlerFunc(handler), middleware...)
}

// retrieveMethods returns all allowed methods of the request
// path. And the result is random, since it uses map.
func (r *Router) retrieveMethods(path string) (methods []string) {
	for method, reg := range r.combinedRegexps {
		if reg.MatchString(path) {
			methods = append(methods, method)
		}
	}

	return
}

// ServeHTTP implements http.Handler's ServeHTTP method.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// handle panic if PanicHandler is set.
	if r.PanicHandler != nil {
		defer func() {
			if rcv := recover(); rcv != nil {
				r.PanicHandler(w, req, rcv)
			}
		}()
	}

	method := req.Method
	path := req.URL.Path
	// fetch group.
	router, path := r.fetchGroup(path)
	if reg, ok := router.combinedRegexps[method]; ok {
		matches := reg.FindStringSubmatch(path)
		if matches != nil {
			// fetch route
			var i = 1
			for ; i < len(matches) && matches[i] == ""; i++ {
			}
			route := router.routes[method][i]

			// handle trailing slashes.
			if r.TrailingSlashesPolicy != IgnoreTrailingSlashes {
				// status code, default 301.
				code := http.StatusMovedPermanently
				if method != http.MethodGet {
					// status code should be 308 if the request is not a GET request.
					code = http.StatusPermanentRedirect
				}

				pos := len(req.URL.Path) - 1
				isRootPath := req.URL.Path == "/"
				endWithSlashes := req.URL.Path[pos] == '/'
				if r.TrailingSlashesPolicy == RemoveTrailingSlashes && endWithSlashes && !isRootPath {
					req.URL.Path = req.URL.Path[:pos]
					http.Redirect(w, req, req.URL.String(), code)
					return
				}
				if r.TrailingSlashesPolicy == AppendTrailingSlashes && !endWithSlashes && !isRootPath {
					req.URL.Path = req.URL.Path + "/"
					http.Redirect(w, req, req.URL.String(), code)
					return
				}
				if r.TrailingSlashesPolicy == StrictTrailingSlashes && !isRootPath {
					if route.hasTrailingSlashes && !endWithSlashes {
						req.URL.Path = req.URL.Path + "/"
						http.Redirect(w, req, req.URL.String(), code)
						return
					}
					if !route.hasTrailingSlashes && endWithSlashes {
						req.URL.Path = req.URL.Path[:pos]
						http.Redirect(w, req, req.URL.String(), code)
					}
				}
			}

			if len(route.params) > 0 {
				// extract parameters from the URL path.
				params := make(map[string]string, len(route.params))
				for _, name := range route.params {
					i++
					params[name] = matches[i]
				}

				// pass parameters to downstream handler via context.
				ctx := context.WithValue(req.Context(), contextParamsKey, params)
				req = req.WithContext(ctx)
			}

			// handle request
			route.finalHandler.ServeHTTP(w, req)
			return
		}
	}

	// retrieve allowed methods
	methods := router.retrieveMethods(path)

	// handle OPTIONS request.
	if method == http.MethodOptions {
		if r.OptionsHandler != nil {
			r.OptionsHandler(w, req, methods)
			return
		}

		w.Header().Set("Allow", strings.Join(methods, ", "))
		return
	}

	// retrieve the allowed methods of the URL path.
	if len(methods) > 0 {
		// handle Method Not Allowed.
		if r.MethodNotAllowedHandler != nil {
			r.MethodNotAllowedHandler(w, req, methods)
			return
		}

		w.Header().Set("Allow", strings.Join(methods, ", "))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// handle Not Found.
	if r.NotFoundHandler != nil {
		r.NotFoundHandler.ServeHTTP(w, req)
		return
	}

	http.NotFound(w, req)
}

func (r *Router) middleware() (middleware []Middleware) {
	middleware = append(r.Middleware, middleware...)

	if r.parent != nil {
		middleware = append(r.parent.middleware(), middleware...)
	}

	return
}

func (r *Router) fetchGroup(path string) (*Router, string) {
	router := r
walk:
	if path != "/" && len(r.groups) > 0 {
		i := 1
		for ; i < len(path) && path[i] != '/'; i++ {
		}
		if i > 1 {
			prefix := path[1:i]
			if group, ok := router.groups[prefix]; ok {
				router = group
				if i < len(path) {
					path = path[i:]
					goto walk
				}
				path = "/"
			}
		}
	}

	return router, path
}

type route struct {
	reg string

	params []string

	hasTrailingSlashes bool

	middleware []Middleware

	handler http.Handler

	finalHandler http.Handler
}

// Middleware is a chaining tool for chaining http.Handler.
//
// Handler workflow:
//      Root Router Middleware
//               ↓
//     Group Router Middleware
//               ↓
//        Handler Middleware
//               ↓
//             Handler
type Middleware func(next http.Handler) http.Handler

// Params returns the parameters of the request path.
func Params(r *http.Request) map[string]string {
	if params, ok := r.Context().Value(contextParamsKey).(map[string]string); ok {
		return params
	}

	return nil
}
