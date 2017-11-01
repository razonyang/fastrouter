package fastrouter_test

import (
	"log"
	"net/http"
	"strings"

	"github.com/razonyang/fastrouter"
)

func Example() {
	r := fastrouter.New()

	// TrailingSlashesPolicy
	r.TrailingSlashesPolicy = fastrouter.StrictTrailingSlashes
	// PanicHandler
	r.PanicHandler = func(w http.ResponseWriter, req *http.Request, rcv interface{}) {
		log.Printf("received a panic: %#v\n", rcv)
	}
	// OptionsHandler
	r.OptionsHandler = func(w http.ResponseWriter, req *http.Request, methods []string) {
		w.Header().Set("Allow", strings.Join(methods, ", "))
		w.Write([]byte("user-defined OptionsHandler"))
	}
	// MethodNotAllowedHandler
	r.MethodNotAllowedHandler = func(w http.ResponseWriter, req *http.Request, methods []string) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Allow", strings.Join(methods, ", "))
		w.Write([]byte("user-defined MethodNotAllowedHandler"))
	}
	// NotFoundHandler
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("user-defined NotFoundHandler"))
	})

	// homepage handler
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})
	// panic handler
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("panic handler")
	})
	// hello handler
	r.Get("/hello/<name>", func(w http.ResponseWriter, r *http.Request) {
		var name string

		name = fastrouter.Params(r)["name"]
		/* The following code snippet is equivalent.
		if params, ok := r.Context().Value(fastrouter.ParamsKey{}).(map[string]string); ok {
			name = params["name"]
		}
		*/

		w.Write([]byte("hello " + name))
	})

	// RESTful API
	r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("query users"))
	})
	r.Post("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("create an user"))
	})
	r.Get("/users/<name>", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fastrouter.Params(r)["name"] + " profile"))
	})
	r.Delete("/users/<name>", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("deleted user " + fastrouter.Params(r)["name"]))
	})
	r.Put("/users/<name>", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("updated " + fastrouter.Params(r)["name"] + " profile"))
	})
	r.Get("/users/<name>/posts", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("the posts created by " + fastrouter.Params(r)["name"]))
	})

	// Make preparations before handling incoming request.
	// Note that, this method MUST be invoked before handling incoming request,
	// otherwise the router can not works as expected.
	r.Prepare()

	log.Fatal(http.ListenAndServe(":8080", r))
}

func ExampleMiddleware() {
	// basic auth middleware
	basicAuthMiddleware := func(next http.Handler) http.Handler {
		username := "foo"
		password := "bar"

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if name, passwd, ok := r.BasicAuth(); ok && name == username && passwd == password {
				// authorized, continue processing
				next.ServeHTTP(w, r)
				return
			}

			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	// body limit middleware
	bodyLimitMiddleware := func(next http.Handler) http.Handler {
		var limit int64 = 1024

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limit {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}

			// continue processing
			next.ServeHTTP(w, r)
		})
	}

	r := fastrouter.New()

	// set basic auth middleware as global middleware.
	r.Middleware = append(r.Middleware, basicAuthMiddleware)

	postCreate := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Created"))
	}
	r.Post(`/posts`, postCreate)

	// set body limit middleware for upload handler.
	upload := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Uploaded"))
	}
	r.Post(`/upload`, upload, bodyLimitMiddleware)

	// Make preparations before handling incoming request.
	// Note that, this method MUST be invoked before handling incoming request,
	// otherwise the router can not works as expected.
	r.Prepare()

	log.Fatal(http.ListenAndServe(":8080", r))
}

func ExampleRouter_Group() {
	r := fastrouter.New()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("root"))
	})

	// group frontend
	frontend := r.Group("frontend")
	frontend.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("frontend"))
	})
	// nested group
	frontendUser := frontend.Group("user")
	frontendUser.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("frontend user"))
	})

	// group backend
	backend := r.Group("backend")
	backend.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend"))
	})

	// Make preparations before handling incoming request.
	// Note that, this method MUST be invoked before handling incoming request,
	// otherwise the router can not works as expected.
	r.Prepare()

	log.Fatal(http.ListenAndServe(":8080", r))
}

func ExampleRouter_ServeFiles() {
	r := fastrouter.New()

	// matched the root directory resources,
	// subdirectory resources can not be matched.
	r.ServeFiles("/css/<filepath>", "/path-to-css/")

	// both of root directory and subdirectory resources
	// cab be matched.
	r.ServeFiles("/js/<filepath:.+>", "/path-to-js/")

	// Make preparations before handling incoming request.
	// Note that, this method MUST be invoked before handling incoming request,
	// otherwise the router can not works as expected.
	r.Prepare()

	log.Fatal(http.ListenAndServe(":8080", r))
}
