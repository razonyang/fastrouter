// Copyright 2017 Razon Yang. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fastrouter

import (
	"fmt"
	"regexp"
)

// ParserInterface defines a Parse method for parsing pattern.
type ParserInterface interface {
	// Parse extracts information from pattern.
	//
	// The pattern will be parsed by parser, parse rule is related to parser.
	//
	// The regexp MUST be a valid regular expression string for
	// indicating which request paths can be matched.
	//
	// The params is a slice that contains pattern named parameters,
	// in order.
	//
	// The hasTrailingSlashes indicate that whether pattern has
	// trailing slashes, this flag has effect on strict trailing
	// slashes policy.
	//
	// Returns non-nil error, if parsing failed.
	Parse(pattern string) (regexp string, params []string, hasTrailingSlashes bool, err error)
}

var defaultParserRegexp = regexp.MustCompile(`<([^/:]+)(:([^/]+))?>`)

// NewParser returns a new parser via NewParserWithReg with the
// defaultParserRegexp.
func NewParser() Parser {
	return NewParserWithReg(defaultParserRegexp)
}

// NewParserWithReg returns a new parser with the given regexp.
func NewParserWithReg(reg *regexp.Regexp) Parser {
	return Parser{reg: reg}
}

// Parser is the default pattern parser which implements
// ParserInterface.
type Parser struct {
	// reg for detecting named parameters and converting
	// pattern into a regexp string.
	reg *regexp.Regexp
}

// Parse implements ParserInterface's Parse method.
//
// The pattern MUST be begin with '/', the pattern parse rule
// is related regexp, by default defaultParserRegexp is used,
// you can also define your own parse rule via NewParserWithReg.
//
// The following introduction and examples is about of defaultParserRegexp.
//
// The pattern can be divided into two types:
//
// 1. Without named parameter:
//     "/"
//     "/users"
//     ...
// 2. With named parameter:
//     `/users/<name>`
//     `/users/<name>/posts`
//     `/posts/<year:\d{4}>/<month:\d{2}>/<title>`
//     ...
// Named parameter MUST be one of '<name>' and '<name:regexp>'.
//     `<name>`        // will be converted to `([^/]+)`
//
//     `<name:regexp>` // will be converted to `(regexp)`
//
// Examples:
//     | Pattern                                     | Error   | Regexp                             | hasTrailingSlashes | Params                               |
//     |:--------------------------------------------|:--------|:-----------------------------------|:-------------------|:-------------------------------------|
//     |                                             | non-nil |                                    |                    |                                      |
//     | `no-start-with-slashes`                     | non-nil |                                    |                    |                                      |
//     | `/`                                         | nil     | `//?`                              | NO                 |                                      |
//     | `/hello/<name>`                             | nil     | `/hello/([^/]+)/?`                 | NO                 | `[]string{"name"}`                   |
//     | `/users`                                    | nil     | `/users/?`                         | NO                 |                                      |
//     | `/users/<name:\w+>`                         | nil     | `/users/(\w+)/?`                   | NO                 | `[]string{"name"}`                   |
//     | `/users/<name:\w+>/posts/`                  | nil     | `/users/(\w+)/posts/?`             | YES                | `[]string{"name"}`                   |
//     | `/orders/<id:\d+>`                          | nil     | `/orders/(\d+)/?`                  | NO                 | `[]string{"id"}`                     |
//     | `/posts/<year:\d{4}>/<month:\d{2}>/<title>` | nil     | `/posts/(\d{4})/(\d{2})/([^/]+)/?` | NO                 | `[]string{"year", "month", "title"}` |
func (p Parser) Parse(pattern string) (regexp string, params []string, hasTrailingSlashes bool, err error) {
	if pattern == "" || pattern[0] != '/' {
		err = fmt.Errorf(`the pattern MUST begin with '/' in pattern %q`, pattern)
		return
	}

	if pattern != "/" && pattern[len(pattern)-1] == '/' {
		hasTrailingSlashes = true
		pattern = pattern[:len(pattern)-1]
	}

	// fetch named parameters.
	matches := p.reg.FindAllStringSubmatch(pattern, -1)
	if matches != nil {
		for _, match := range matches {
			params = append(params, match[1])
		}

		// convert pattern into a regexp string.
		i := -1
		regexp = p.reg.ReplaceAllStringFunc(pattern, func(any string) string {
			i++
			if matches[i][3] != "" {
				return "(" + matches[i][3] + ")"
			}

			return `([^/]+)`
		})
	} else {
		regexp = pattern
	}

	regexp += "/?"

	return
}
