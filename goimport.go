package main // import "arp242.net/goimport"

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

const stdin = "<standard input>"

// StringList appends strings.
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
	add, rm, sub      StringList
	write, get, force bool
}

func main() {
	var opts options

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: goimport [flags] [path ...]\n")
		flag.PrintDefaults()
		os.Exit(2)
	}

	flag.BoolVar(&opts.get, "g", false, "attempt to 'go get' packages not in GOPATH")
	flag.BoolVar(&opts.force, "f", false, "force adding/removing package, unless there is a fatal error")
	flag.BoolVar(&opts.write, "w", false, "write result to (source) file instead of stdout")
	flag.Var(&opts.add, "add", "add import; can be given multiple times")
	flag.Var(&opts.sub, "sub", "like -add, but if an existing package with this name exists it will be replaced")
	flag.Var(&opts.rm, "rm", "remove import; can be given multiple times")
	flag.Parse()
	paths := flag.Args()

	if len(opts.add)+len(opts.sub)+len(opts.rm) == 0 {
		fatal(errors.New("need at least one -add, -sub, or -rm"))
	}

	switch len(paths) {

	case 0:
		if opts.write {
			fatal(errors.New("can't use -w with stdin"))
		}

		//fmt.Fprintf(os.Stderr, "goimport: reading from stdin...\n")
		if err := process(stdin, opts); err != nil {
			fatal(err)
		}

	case 1:
		if err := process(paths[0], opts); err != nil {
			fatal(err)
		}

	default:
		if !opts.write {
			fatal(errors.New("need to use -w with multiple files"))
		}

		for _, path := range paths {
			if err := process(path, opts); err != nil {
				fatal(err)
			}
		}
	}
}

// process a file.
func process(filename string, opts options) error {
	var in io.Reader
	if filename == stdin {
		in = os.Stdin
	} else {
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
		if _, err := os.Stdout.Write(res); err != nil {
			return err
		}
	}

	return nil
}

func rewrite(filename string, src []byte, opts options) ([]byte, error) {
	fset := token.NewFileSet()
	//file, err := parser.ParseFile(fset, filename, src, parser.ImportsOnly)
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	imports := []string{}
	importsBase := []string{}
	for _, imp := range astutil.Imports(fset, file) {
		for _, i := range imp {
			imports = append(imports, strings.Trim(i.Path.Value, `"`))
			importsBase = append(importsBase, path.Base(strings.Trim(i.Path.Value, `"`)))
		}
	}

	for _, pkg := range opts.add {
		if err := addPackage(fset, file, pkg, opts, false); err != nil {
			return nil, err
		}
	}
	for _, pkg := range opts.sub {
		if err := addPackage(fset, file, pkg, opts, true); err != nil {
			return nil, err
		}
	}

	for _, pkg := range opts.rm {
		// TODO: deal with named imports.
		pkg = strings.Trim(pkg, `"/`)
		astutil.DeleteImport(fset, file, pkg)
	}

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

func addPackage(fset *token.FileSet, file *ast.File, pkg string, opts options, sub bool) error {

	imports := []string{}
	importsBase := []string{}
	for _, imp := range astutil.Imports(fset, file) {
		for _, i := range imp {
			imports = append(imports, strings.Trim(i.Path.Value, `"`))
			importsBase = append(importsBase, path.Base(strings.Trim(i.Path.Value, `"`)))
		}
	}

	pkgAlias := ""
	if strings.Contains(pkg, ":") {
		s := strings.Split(pkg, ":")
		if len(s) != 2 {
			return fmt.Errorf("invalid package name: %v", s)
		}

		pkg = s[0]
		pkgAlias = s[1]
	}

	if !opts.force && InStringSlice(imports, pkg) {
		return fmt.Errorf("import '%v' is already used", pkg)
	}

	for _, imp := range imports {
		if path.Base(imp) != path.Base(pkg) {
			continue
		}

		if sub {
			astutil.DeleteImport(fset, file, imp)
		} else if !opts.force {
			return fmt.Errorf("import '%v' would conflict", pkg)
		}

	}

	pkg = strings.Trim(pkg, `"/`)
	if !exists(pkg) {
		if opts.get {
			// TODO
		}

		if !opts.force {
			return fmt.Errorf("import '%v' is not in GOPATH", pkg)
		}
	}

	if pkgAlias == "" {
		astutil.AddImport(fset, file, pkg)
	} else {
		astutil.AddNamedImport(fset, file, pkg, pkgAlias)
	}

	return nil
}

// InStringSlice reports whether str is within list
func InStringSlice(list []string, str string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}
	return false
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "goimport: %v\n", err)
	os.Exit(1)
}

func exists(pkg string) bool {
	cmd := exec.Command("go", "list", pkg)
	return cmd.Run() == nil
}
