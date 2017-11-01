// Copyright 2017 Razon Yang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fastrouter

import (
	"fmt"
	"reflect"
	"testing"
)

type testPattern struct {
	reg                string
	params             []string
	hasTrailingSlashes bool
	err                error
}

func TestPatternParser_Parse(t *testing.T) {
	emptyParams := []string{}
	testPatterns := map[string]testPattern{
		"": {"",
			emptyParams,
			false,
			fmt.Errorf(`the pattern MUST begin with '/' in pattern %q`, ""),
		},
		"/": {"//?",
			emptyParams,
			false,
			nil,
		},
		"users": {"",
			emptyParams,
			false,
			fmt.Errorf(`the pattern MUST begin with '/' in pattern %q`, "users"),
		},
		`/users`:          {"/users/?", emptyParams, false, nil},
		`/users/`:         {"/users/?", emptyParams, true, nil},
		`/users/<id>`:     {"/users/([^/]+)/?", []string{"id"}, false, nil},
		`/users/<id:\d+>`: {`/users/(\d+)/?`, []string{"id"}, false, nil},
		`/posts/<year:\d{4}>/<month:\d{2}>/<title>`: {
			`/posts/(\d{4})/(\d{2})/([^/]+)/?`,
			[]string{"year", "month", "title"},
			false,
			nil,
		},
	}

	parser := NewParser()
	for pattern, v := range testPatterns {
		reg, params, hasTrailingSlashes, err := parser.Parse(pattern)
		if v.reg != reg {
			t.Errorf("expect the reg of pattern %q to be %q, but got %q", pattern, v.reg, reg)
		}
		if !compareSlice(v.params, params) {
			t.Errorf("expect the params of pattern %q to be %v, but got %v", pattern, v.params, params)
		}
		if v.hasTrailingSlashes != hasTrailingSlashes {
			t.Errorf("expect the hasTrailingSlashes of pattern %q to be %v, but got %v", pattern, v.hasTrailingSlashes, hasTrailingSlashes)
		}
		if !reflect.DeepEqual(v.err, err) {
			t.Errorf("expect the err of pattern %q to be %v, but got %v", pattern, v.err, err)
		}
	}
}
