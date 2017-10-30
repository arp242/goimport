package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	cases := []struct {
		in   string
		opts options
		want string
	}{
		{
			`package main`,
			options{},
			`package main
`,
		},
		{
			`package main`,
			options{add: StringList{"errors"}},
			`package main

import "errors"
`,
		},
		{
			`package main`,
			options{add: StringList{"errors", "fmt"}},
			`package main

import (
	"errors"
	"fmt"
)
`,
		},

		{
			`package main`,
			options{rm: StringList{"errors", "fmt"}},
			`package main
`,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			buf := bytes.NewBuffer([]byte{})
			err := process("test", strings.NewReader(tc.in), buf, tc.opts)
			if err != nil {
				t.Fatal(err)
			}

			out := buf.String()
			if out != tc.want {
				t.Errorf("\nout:  %#v\nwant: %#v\n", out, tc.want)
			}
		})
	}
}
