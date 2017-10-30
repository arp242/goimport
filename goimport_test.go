package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestRewrite(t *testing.T) {
	cases := []struct {
		opts              options
		in, want, wantErr string
	}{
		{
			options{},
			"package main\n",
			"package main\n",
			"",
		},
		{
			options{add: StringList{"errors"}},
			"package main\n",
			"package main\n\nimport \"errors\"\n",
			"",
		},
		{
			options{add: StringList{`"errors/"`}},
			"package main\n",
			"package main\n\nimport \"errors\"\n",
			"",
		},
		{
			options{add: StringList{"errors", "fmt"}},
			"package main\n",
			`
				package main

				import (
					"errors"
					"fmt"
				)
			`,
			"",
		},

		{
			options{rm: StringList{"errors", "fmt"}},
			"package main\n",
			"package main\n",
			"",
		},
		{
			options{rm: StringList{"errors", "fmt"}},
			"package main\n\nimport \"errors\"",
			"package main\n",
			"",
		},

		{
			options{add: StringList{"io"}, rm: StringList{"errors", "fmt"}},
			`
				package main

				import (
					"errors"
					"fmt"
				)

				// comment
				var a = "x"
			`,
			`
				package main

				import (
					"io"
				)

				// comment
				var a = "x"
			`,
			"",
		},
		{
			options{add: StringList{"errors"}},
			"package main\n\nimport \"errors\"\n",
			"",
			"import 'errors' is already used",
		},
		{
			options{add: StringList{"text/template"}},
			"package main\n\nimport \"html/template\"\n",
			"",
			"import 'text/template' would conflict",
		},

		{
			options{add: StringList{"does/not/exist"}},
			"package main\n",
			"",
			"import 'does/not/exist' is not in GOPATH",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			outB, err := rewrite("test", []byte(tc.in), tc.opts)
			if !ErrorContains(err, tc.wantErr) {
				t.Fatalf("wrong error\nout:  %#v\nwant: %#v\n", err, tc.wantErr)
			}

			tc.want = normalizeSpace(tc.want)
			out := normalizeSpace(string(outB))

			if out != tc.want {
				t.Errorf("\nout:  %#v\nwant: %#v\n", out, tc.want)
			}
		})
	}
}

func normalizeSpace(in string) string {
	indent := 0
	for i := 0; i < len(in); i++ {
		switch in[i] {
		case '\n':
			// Do nothing
		case '\t':
			indent++
		default:
			break
		}
	}

	r := ""
	for _, line := range strings.Split(in, "\n") {
		r += strings.Replace(line, "\t", "", indent) + "\n"
	}

	return strings.TrimSpace(r)
}

// ErrorContains checks if the error message in got contains the text in
// expected.
//
// This is safe when got is nil. Use an empty string for expected if you want to
// test that err is nil.
func ErrorContains(got error, expected string) bool {
	if got == nil {
		return expected == ""
	}
	if expected == "" {
		return false
	}
	return strings.Contains(got.Error(), expected)
}
