package main // import "arp242.net/goimport"

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// StringList appends eachs List of strings.
type StringList []string

func (l *StringList) String() string {
	return strings.Join(*l, ", ")
}

// Set the value.
func (l *StringList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

type options struct {
	add   StringList
	rm    StringList
	write bool
}

func main() {
	var opts options

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: goimport [flags] [path ...]\n")
		flag.PrintDefaults()
		os.Exit(2)
	}

	write := flag.Bool("w", false, "write result to (source) file instead of stdout")
	flag.Var(&opts.add, "add", "Add import; can be given multiple times")
	flag.Var(&opts.rm, "rm", "Remove import; can be given multiple times")

	flag.Parse()
	opts.write = *write
	paths := flag.Args()

	if len(opts.add) == 0 && len(opts.rm) == 0 {
		fatal(errors.New("need at least one -add or -rm"))
	}

	switch len(paths) {

	case 0:
		if opts.write {
			fatal(errors.New("can't use -w with stdin"))
		}

		//fmt.Fprintf(os.Stderr, "goimport: reading from stdin...\n")
		if err := process("<standard input>", os.Stdin, os.Stdout, opts); err != nil {
			fatal(err)
		}

	case 1:
		if err := process(paths[0], nil, os.Stdout, opts); err != nil {
			fatal(err)
		}

	default:
		if !opts.write {
			fatal(errors.New("need to use -w with multiple files"))
		}

		for _, path := range paths {
			if err := process(path, nil, os.Stdout, opts); err != nil {
				fatal(err)
			}
		}
	}
}

func process(filename string, in io.Reader, out io.Writer, opts options) error {
	if in == nil {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close() // nolint: errcheck
		in = f
	}

	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	res, err := rewrite(filename, src, opts)
	if err != nil {
		return err
	}

	if opts.write {
		if err = ioutil.WriteFile(filename, res, 0); err != nil {
			return err
		}
	} else {
		if _, err := out.Write(res); err != nil {
			return err
		}
	}

	return nil
}

func rewrite(filename string, src []byte, opts options) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, add := range opts.add {
		if strings.Contains(add, ":") {
			s := strings.Split(add, ":")
			if len(s) != 2 {
				return nil, fmt.Errorf("invalid -add: %v", s)
			}
			astutil.AddNamedImport(fset, file, s[1], s[0])
		} else {
			astutil.AddImport(fset, file, add)
		}
	}
	for _, rm := range opts.rm {
		// TODO: deal with named imports.
		astutil.DeleteImport(fset, file, rm)
	}

	// TODO: what to do if a package already exists?

	// Write output.
	printConfig := &printer.Config{}
	var buf bytes.Buffer
	err = printConfig.Fprint(&buf, fset, file)
	if err != nil {
		return nil, err
	}

	out, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return out, nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "goimport: %v\n", err)
	os.Exit(1)
}
