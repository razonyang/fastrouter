// Copyright 2017 Razon Yang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fastrouter

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
)

func TestRouter_Group(t *testing.T) {
	r := New()
	if len(r.groups) > 0 {
		t.Errorf("expect no groups, but got %v", r.groups)
	}

	prefix := "v1"
	v1 := r.Group(prefix)
	if !reflect.DeepEqual(v1.parser, r.parser) {
		t.Errorf("expect parser of v1 to be %v, but got %v", r.parser, v1.parser)
	}
	if group, ok := r.groups[prefix]; !ok || !reflect.DeepEqual(group, v1) {
		t.Errorf("expect group v1 to be %v, but got %v", v1, group)
	}

	prefix2 := "v2"
	v2 := r.Group(prefix2)
	if group, ok := r.groups[prefix2]; !ok || !reflect.DeepEqual(group, v2) {
		t.Errorf("expect group v2 to be %v, but got %v", v2, group)
	}

	expect := fmt.Errorf("the group which prefix equal to %q already exists", prefix)
	defer func() {
		if rcv := recover(); rcv == nil || !reflect.DeepEqual(expect, rcv) {
			t.Errorf("expect err to be %q, but got %q", expect, rcv)
		}
	}()
	r.Group(prefix)
}

func TestRouter_Group2(t *testing.T) {
	expect := `the group prefix MUST NOT be empty`
	defer func() {
		if rcv := recover(); rcv == nil || !reflect.DeepEqual(expect, rcv) {
			t.Errorf("expect err to be %q, but got %q", expect, rcv)
		}
	}()

	r := New()
	r.Group("")
}

func TestRouter_Group3(t *testing.T) {
	expect := `the group prefix MUST NOT contains '/'`
	defer func() {
		if rcv := recover(); rcv == nil || !reflect.DeepEqual(expect, rcv) {
			t.Errorf("expect err to be %q, but got %q", expect, rcv)
		}
	}()

	r := New()
	r.Group("/v1")
}

func TestRouter_Group4(t *testing.T) {
	r := New()
	r.Get("/", helloHandler("hello world"))
	v1 := r.Group("v1")
	v1Msg := "group v1"
	v1.Get("/", helloHandler(v1Msg))

	v2 := r.Group("v2")
	v2Msg := "group v2"
	v2.Get("/", helloHandler(v2Msg))
	v2.Get("/users", helloHandler("v2 users"))

	r.Prepare()

	var req *http.Request
	var w *httptest.ResponseRecorder
	var body string

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	body = "hello world"
	if w.Body.String() != body {
		t.Errorf("expect response body to be %q, but got %q", body, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != v1Msg {
		t.Errorf("expect response body to be %q, but got %q", v1Msg, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v2", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != v2Msg {
		t.Errorf("expect response body to be %q, but got %q", v2Msg, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v2/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	body = "v2 users"
	if w.Body.String() != body {
		t.Errorf("expect response body to be %q, but got %q", body, w.Body.String())
	}
}

func TestRouter_Handle(t *testing.T) {
	defer func() {
		if rcv := recover(); rcv == nil {
			t.Errorf("expect rcv is not nil, but got %v", rcv)
		}
	}()
	r := New()
	r.Handle(http.MethodGet, "", emptyHandler)
}

func TestParams(t *testing.T) {
	r := New()
	var params map[string]string
	r.Get(`/users`, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params = Params(r)
	}))
	r.Get(`/users/<id:\d+>`, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params = Params(r)
	}))
	r.Get(`/users/<id:\d+>/posts/<title>`, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params = Params(r)
	}))
	r.Prepare()

	var expect interface{}

	req := httptest.NewRequest(http.MethodGet, `/users`, nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	if len(params) != 0 {
		t.Errorf("expect params to be empty, but got %#v", params)
	}

	req = httptest.NewRequest(http.MethodGet, `/users/11`, nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	expect = map[string]string{"id": "11"}
	if !reflect.DeepEqual(params, expect) {
		t.Errorf("expect params to be %v, but got %v", expect, params)
	}

	req = httptest.NewRequest(http.MethodGet, `/users/22/posts/hello`, nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	expect = map[string]string{"id": "22", "title": "hello"}
	if !reflect.DeepEqual(params, expect) {
		t.Errorf("expect params to be %v, but got %v", expect, params)
	}
}

func TestRouter_RetrieveMethods(t *testing.T) {
	r := New()
	r.Prepare()

	path := `/users/1`
	if methods := r.retrieveMethods(path); len(methods) > 0 {
		t.Errorf("expect no allowed methods, but got %v", methods)
	}

	pattern := `/users/<id>`
	r.Get(pattern, emptyHandler)
	r.Prepare()
	expect := []string{http.MethodGet}
	if methods := r.retrieveMethods(path); !compareSlice(expect, methods) {
		t.Errorf("expect method to be %v, but got %v", expect, methods)
	}

	r.Delete(pattern, emptyHandler)
	r.Put(pattern, emptyHandler)
	r.Prepare()
	expect = []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	if methods := r.retrieveMethods(path); !compareSlice(expect, methods) {
		t.Errorf("expect method to be %v, but got %v", expect, methods)
	}
}

func TestRouter_PanicHandler(t *testing.T) {
	r := New()
	err := "panic message"
	r.PanicHandler = func(w http.ResponseWriter, req *http.Request, rcv interface{}) {
		if !reflect.DeepEqual(rcv, err) {
			t.Errorf("expect panic to be %q, but got %q", err, rcv)
		}
	}
	r.Get(`/panic`, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(err)
	}))
	r.Prepare()

	req := httptest.NewRequest(http.MethodGet, `/panic`, nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
}

// Default OptionsHandler
func TestRouter_OptionsHandler(t *testing.T) {
	r := New()
	r.Delete(`/users`, emptyHandler)
	r.Prepare()

	req := httptest.NewRequest(http.MethodOptions, `/users`, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	if !reflect.DeepEqual(w.Header().Get("Allow"), "DELETE") {
		t.Errorf("expect header's Allow to be %q, but got %q", w.Header().Get("Allow"), "DELETE")
	}
}

// Specify OptionsHandler
func TestRouter_OptionsHandler2(t *testing.T) {
	r := New()
	r.Delete(`/users`, emptyHandler)
	originKey := "Access-Control-Allow-Origin"
	methodsKey := "Access-Control-Allow-Methods"
	r.OptionsHandler = func(w http.ResponseWriter, r *http.Request, methods []string) {
		w.Header().Set(originKey, "*")
		w.Header().Set(methodsKey, strings.Join(methods, ", "))
	}
	r.Prepare()

	req := httptest.NewRequest(http.MethodOptions, `/users`, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	if !reflect.DeepEqual(w.Header().Get(originKey), "*") {
		t.Errorf("expect header's %q to be %q, but got %q", originKey, "*", w.Header().Get(originKey))
	}
	if !reflect.DeepEqual(w.Header().Get(methodsKey), "DELETE") {
		t.Errorf("expect header's %q to be %q, but got %q", methodsKey, "DELETE", w.Header().Get(methodsKey))
	}
}

// Default MethodNotAllowedHandler
func TestRouter_MethodNotAllowedHandler(t *testing.T) {
	r := New()
	r.Post(`/users`, emptyHandler)
	r.Prepare()

	req := httptest.NewRequest(http.MethodGet, `/users`, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expect status code to be %d, but got %d", http.StatusMethodNotAllowed, w.Code)
	}
	if !reflect.DeepEqual(w.Header().Get("Allow"), "POST") {
		t.Errorf("expect header's Allow to be %q, but got %q", w.Header().Get("Allow"), "POST")
	}
}

// Specify MethodNotAllowedHandler
func TestRouter_MethodNotAllowedHandler2(t *testing.T) {
	r := New()
	r.Post(`/users`, emptyHandler)
	body := "405 Method Not Allowed"
	r.MethodNotAllowedHandler = func(w http.ResponseWriter, r *http.Request, methods []string) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Allow", strings.Join(methods, ", "))
		w.Write([]byte(body))
	}
	r.Prepare()

	req := httptest.NewRequest(http.MethodGet, `/users`, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expect status code to be %d, but got %d", http.StatusMethodNotAllowed, w.Code)
	}
	if !reflect.DeepEqual(w.Header().Get("Allow"), "POST") {
		t.Errorf("expect header's Allow to be %q, but got %q", w.Header().Get("Allow"), "POST")
	}
	if w.Body.String() != body {
		t.Errorf("expect response body to be %q, but got %q", body, w.Body.String())
	}
}

// Default NotFoundHandler
func TestRouter_NotFoundHandler(t *testing.T) {
	r := New()
	r.Prepare()

	req := httptest.NewRequest(http.MethodGet, `/not-found`, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expect status code to be %d, but got %d", http.StatusNotFound, w.Code)
	}
}

// Specify NotFoundHandler
func TestRouter_NotFoundHandler2(t *testing.T) {
	r := New()
	body := "404 Not Found"
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(body))
	})
	r.Prepare()

	req := httptest.NewRequest(http.MethodGet, `/not-found`, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expect status code to be %d, but got %d", http.StatusNotFound, w.Code)
	}
	if w.Body.String() != body {
		t.Errorf("expect response body to be %q, but got %q", body, w.Body.String())
	}
}

func TestRouter_ServeFiles(t *testing.T) {
	expect := `the pattern MUST contains parameter placeholder named "filepath"`
	defer func() {
		if rcv := recover(); rcv == nil || !reflect.DeepEqual(expect, rcv) {
			t.Errorf("expect err to be %q, but got %q", expect, rcv)
		}
	}()

	r := New()
	r.ServeFiles("/tmp", os.TempDir())
	r.Prepare()
}

func TestRouter_ServeFiles2(t *testing.T) {
	r := New()
	r.ServeFiles("/tmp/<filepath:.+>", os.TempDir())
	r.Prepare()

	tmpFile := "fastrouter_tmp_files"
	data := []byte("TestRouter_ServeFiles2")
	if err := ioutil.WriteFile(path.Join(os.TempDir(), tmpFile), data, 0666); err != nil {
		t.Fatalf("failed to create tmp file: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tmp/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expect status code to be %d, but got %d", http.StatusNotFound, w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/tmp/"+tmpFile, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusNotFound, w.Code)
	}
}

// Ignore trailing slashes
func TestRouter_TrailingSlashesPolicy(t *testing.T) {
	r := New()
	r.Get("/users", emptyHandler)
	r.Prepare()

	r.TrailingSlashesPolicy = IgnoreTrailingSlashes
	var req *http.Request
	var w *httptest.ResponseRecorder

	req = httptest.NewRequest(http.MethodGet, "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/users/", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
}

// Append trailing slashes
func TestRouter_TrailingSlashesPolicy2(t *testing.T) {
	r := New()
	r.Get("/users", emptyHandler)
	r.Get("/users/", emptyHandler)
	r.Post("/users", emptyHandler)
	r.Prepare()

	r.TrailingSlashesPolicy = AppendTrailingSlashes
	var req *http.Request
	var w *httptest.ResponseRecorder
	var location string

	req = httptest.NewRequest(http.MethodGet, "/users/", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expect status code to be %d, but got %d", http.StatusMovedPermanently, w.Code)
	}
	location = "/users/"
	if !reflect.DeepEqual(w.Header().Get("Location"), location) {
		t.Errorf("expect header Location to be %v, but got %v", w.Header().Get("Location"), location)
	}

	req = httptest.NewRequest(http.MethodPost, "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusPermanentRedirect {
		t.Errorf("expect status code to be %d, but got %d", http.StatusPermanentRedirect, w.Code)
	}
	location = "/users/"
	if !reflect.DeepEqual(w.Header().Get("Location"), location) {
		t.Errorf("expect header Location to be %v, but got %v", w.Header().Get("Location"), location)
	}
}

// Remove trailing slashes
func TestRouter_TrailingSlashesPolicy3(t *testing.T) {
	r := New()
	r.Get("/users", emptyHandler)
	r.Get("/users/", emptyHandler)
	r.Post("/users/", emptyHandler)
	r.Prepare()

	r.TrailingSlashesPolicy = RemoveTrailingSlashes
	var req *http.Request
	var w *httptest.ResponseRecorder
	var location string

	req = httptest.NewRequest(http.MethodGet, "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/users/", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expect status code to be %d, but got %d", http.StatusMovedPermanently, w.Code)
	}
	location = "/users"
	if !reflect.DeepEqual(w.Header().Get("Location"), location) {
		t.Errorf("expect header Location to be %v, but got %v", w.Header().Get("Location"), location)
	}

	req = httptest.NewRequest(http.MethodPost, "/users/", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusPermanentRedirect {
		t.Errorf("expect status code to be %d, but got %d", http.StatusPermanentRedirect, w.Code)
	}
	location = "/users"
	if !reflect.DeepEqual(w.Header().Get("Location"), location) {
		t.Errorf("expect header Location to be %v, but got %v", w.Header().Get("Location"), location)
	}
}

// Strict trailing slashes
func TestRouter_TrailingSlashesPolicy4(t *testing.T) {
	r := New()
	r.Get("/users", emptyHandler)
	r.Post("/users/", emptyHandler)
	r.Prepare()

	r.TrailingSlashesPolicy = StrictTrailingSlashes
	var req *http.Request
	var w *httptest.ResponseRecorder
	var location string

	req = httptest.NewRequest(http.MethodGet, "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/users/", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expect status code to be %d, but got %d", http.StatusMovedPermanently, w.Code)
	}
	location = "/users"
	if !reflect.DeepEqual(w.Header().Get("Location"), location) {
		t.Errorf("expect header Location to be %v, but got %v", w.Header().Get("Location"), location)
	}

	req = httptest.NewRequest(http.MethodPost, "/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusPermanentRedirect {
		t.Errorf("expect status code to be %d, but got %d", http.StatusPermanentRedirect, w.Code)
	}
	location = "/users/"
	if !reflect.DeepEqual(w.Header().Get("Location"), location) {
		t.Errorf("expect header Location to be %v, but got %v", w.Header().Get("Location"), location)
	}
}

func TestRouter_Middleware(t *testing.T) {
	middlewareKey := "Middleware"
	anotherMiddlewareKey := "Another-Middleware"
	bodyLimitMiddlewareKey := "Body-Limit-Middleware"

	r := New()
	r.Middleware = append(r.Middleware, newHeaderMiddleware(middlewareKey, "Root"))
	r.Get("/", emptyHandler)

	r.Post("/upload", emptyHandler, newHeaderMiddleware(bodyLimitMiddlewareKey, "upload"))

	v1 := r.Group("v1")
	v1.Middleware = append(v1.Middleware, newHeaderMiddleware(anotherMiddlewareKey, "V1"))
	v1.Get("/", emptyHandler)

	// replace header Middleware
	v2 := r.Group("v2")
	v2.Middleware = append(v2.Middleware, newHeaderMiddleware(middlewareKey, "V2"))
	v2.Get("/", emptyHandler)

	r.Prepare()

	var req *http.Request
	var w *httptest.ResponseRecorder
	var expect string

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	expect = "Root"
	if w.Header().Get(middlewareKey) != expect {
		t.Errorf("expect header %q to be %q, but got %q", middlewareKey, expect, w.Header().Get(middlewareKey))
	}

	req = httptest.NewRequest(http.MethodPost, "/upload", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	expect = "Root"
	if w.Header().Get(middlewareKey) != expect {
		t.Errorf("expect header %q to be %q, but got %q", middlewareKey, expect, w.Header().Get(middlewareKey))
	}
	expect = "upload"
	if w.Header().Get(bodyLimitMiddlewareKey) != expect {
		t.Errorf("expect header %q to be %q, but got %q", bodyLimitMiddlewareKey, expect, w.Header().Get(bodyLimitMiddlewareKey))
	}

	req = httptest.NewRequest(http.MethodGet, "/v1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	expect = "Root"
	if w.Header().Get(middlewareKey) != expect {
		t.Errorf("expect header %q to be %q, but got %q", middlewareKey, expect, w.Header().Get(middlewareKey))
	}
	expect = "V1"
	if w.Header().Get(anotherMiddlewareKey) != expect {
		t.Errorf("expect header %q to be %q, but got %q", anotherMiddlewareKey, expect, w.Header().Get(middlewareKey))
	}

	req = httptest.NewRequest(http.MethodGet, "/v2", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expect status code to be %d, but got %d", http.StatusOK, w.Code)
	}
	expect = "V2"
	if w.Header().Get(middlewareKey) != expect {
		t.Errorf("expect header %q to be %q, but got %q", middlewareKey, expect, w.Header().Get(middlewareKey))
	}
}

func emptyHandler(w http.ResponseWriter, r *http.Request) {}

func helloHandler(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(msg))
	}
}

func newHeaderMiddleware(k, v string) Middleware {
	return func(next http.Handler) http.Handler {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(k, v)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(handler)
	}
}

func compareSlice(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	if len(s1) == 0 {
		return true
	}

	m := make(map[string]bool, len(s1))
	for _, v := range s1 {
		m[v] = true
	}

	for _, v := range s2 {
		if _, ok := m[v]; !ok {
			return false
		}
	}

	return true
}
