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
		// Add
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
			options{add: StringList{`"errors/"`}},
			"package main\nimport ()",
			"package main\n\nimport \"errors\"\n",
			"",
		},

		// rm
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

		// Add and rm
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

		// Replace
		{
			options{replace: StringList{"text/template"}},
			"package main\n\nimport \"html/template\"\n",
			"package main\n\nimport \"text/template\"\n",
			"",
		},

		// Errors
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
			options{add: StringList{"fmt"}, json: true},
			"Not valid Go code.",
			"",
			"ast parse error",
		},
		{
			options{add: StringList{"does/not/exist"}},
			"package main\n",
			"",
			"import 'does/not/exist' is not in GOPATH",
		},
		{
			options{add: StringList{"too:many:colons"}},
			"package main\n",
			"",
			"invalid package name",
		},
		{
			options{add: StringList{"errors:"}},
			"package main\n",
			"",
			"invalid package name",
		},
		{
			options{add: StringList{":errors"}},
			"package main\n",
			"",
			"invalid package name",
		},

		// Comments
		{
			options{add: StringList{"fmt", "time"}},
			"package main\n\nimport \"html/template\" // A comment!\n",
			"package main\n\nimport (\n\"fmt\"\n\"html/template\" // A comment!\n\"time\"\n)\n",
			"",
		},
		{
			options{add: StringList{"fmt", "time"}},
			"package main\n\nimport \"html/template\" /* A comment! */\n",
			"package main\n\nimport (\n\"fmt\"\n\"html/template\" /* A comment! */\n\"time\"\n)\n",
			"",
		},
		{
			options{add: StringList{"fmt", "time"}},
			"package main\n\nimport \"html/template\" /* A */ /* comment! */\n",
			"package main\n\nimport (\n\"fmt\"\n\"html/template\" /* A */ /* comment! */\n\"time\"\n)\n",
			"",
		},
		{
			options{add: StringList{"time"}},
			`
				package main

				import (
					"errors"
					// Commenting fmt
					"fmt"
				)
			`,
			`
				package main

				import (
					"errors"
					"time"
					// Commenting fmt
					"fmt"
				)
			`,
			"",
		},

		// Named imports
		{
			options{add: StringList{"errors:e"}},
			"package main\n",
			"package main\n\nimport e \"errors\"\n",
			"",
		},
		{
			options{rm: StringList{"errors:e"}},
			"package main\n\nimport e \"errors\"\n",
			"package main\n",
			"",
		},
		{
			options{replace: StringList{"html/template:t"}},
			"package main\n\nimport t \"text/template\"\n",
			"package main\n\nimport t \"html/template\"\n",
			"",
		},

		// Preserve blank lines for grouping.
		{
			options{add: StringList{"io"}, rm: StringList{"strings"}},
			`
				package main

				import (
					"errors"

					"fmt"
					"strings"
				)

				// comment
				func main() { }
			`,
			`
				package main

				import (
					"errors"
					"io"

					"fmt"
				)

				// comment
				func main() {}
			`,
			"",
		},

		// JSON
		{
			options{add: StringList{"fmt"}, json: true},
			"package main\nfunc main() { }\n",
			`{"start":0,"end":0,"linedelta":1,"code":"import \"fmt\""}`,
			"",
		},
		{
			options{add: StringList{"fmt"}, json: true},
			"package main\nimport \"errors\"\nfunc main() { }\n",
			`{"start":14,"end":29,"linedelta":1,"code":"import (\n\t\"errors\"\n\t\"fmt\"\n)\n\n"}`,
			"",
		},
		{
			options{add: StringList{`"errors/"`}, json: true},
			"package main\nimport ()",
			`{"start":0,"end":0,"linedelta":1,"code":"import \"errors\""}`,
			"",
		},
		{
			options{json: true, add: StringList{"io"}},
			`
package main

import (
	"fmt"
	"errors"

	"strings"
)

// comment
func main() { }
			`,
			`{"start":16,"end":55,"linedelta":1,"code":"import (\n\t\"fmt\"\n\t\"errors\"\n\n\t\"strings\"\n\t\"io\"\n)\n\n"}`,
			"",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			outB, err := rewrite("test", []byte(tc.in), tc.opts)
			if !errorContains(err, tc.wantErr) {
				t.Fatalf("wrong error\nout:  %v\nwant: %v\n", err, tc.wantErr)
			}

			tc.want = normalizeSpace(tc.want)
			out := normalizeSpace(string(outB))
			if out != tc.want {
				t.Errorf("\nout:  %v\nwant: %v\n", out, tc.want)
			}

			// Make sure -json has the same code.
			/*
				if !tc.opts.json && tc.wantErr == "" {
					t.Run("json", func(t *testing.T) {
						tc.opts.json = true
						outB, err := rewrite("test", []byte(tc.in), tc.opts)
						if !errorContains(err, tc.wantErr) {
							t.Fatalf("wrong error\nout:  %v\nwant: %v\n", err, tc.wantErr)
						}

						j := struct{ Code string }{}
						err = json.Unmarshal(outB, &j)
						if err != nil {
							t.Fatal(err)
						}
						out := normalizeSpace(j.Code)

						// Remove package and code
						want = strings.TrimSpace(want)

						if out != want {
							t.Errorf("\nout:  %#v\nwant: %#v\n", out, want)
						}
					})
				}
			*/
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
		}
	}

	r := ""
	for _, line := range strings.Split(in, "\n") {
		r += strings.Replace(line, "\t", "", indent) + "\n"
	}

	return strings.TrimSpace(r)
}

// errorContains checks if the error message in got contains the text in
// expected.
//
// This is safe when got is nil. Use an empty string for expected if you want to
// test that err is nil.
func errorContains(got error, expected string) bool {
	if got == nil {
		return expected == ""
	}
	if expected == "" {
		return false
	}
	return strings.Contains(got.Error(), expected)
}
