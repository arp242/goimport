[![This project is considered stable](https://img.shields.io/badge/Status-stable-green.svg)](https://arp242.net/status/stable)
[![GoDoc](https://godoc.org/arp242.net/goimport?status.svg)](https://godoc.org/arp242.net/goimport)
[![Go Report Card](https://goreportcard.com/badge/github.com/Carpetsmoker/goimport)](https://goreportcard.com/report/github.com/Carpetsmoker/goimport)
[![Build Status](https://travis-ci.org/Carpetsmoker/goimport.svg?branch=master)](https://travis-ci.org/Carpetsmoker/goimport)
[![codecov](https://codecov.io/gh/Carpetsmoker/goimport/branch/master/graph/badge.svg)](https://codecov.io/gh/Carpetsmoker/goimport)

`goimport` is a tool to add, remove, or replace imports in Go files.

Example usage:

	# Add errors package.
	$ goimport -add errors foo.go

	# Remove errors package.
	$ goimport -rm errors foo.go

	# Add errors package aliased as "errs"
	$ goimport -add errors:errs foo.go

	# Either add an import or replace existing errors with
	# github.com/pkg/errors.
	$ goimport -sub github.com/pkg/errors foo.go

TODO:

- Make `-rm` deal with named imports.
- Make `-sub` work
- Add automatic `go get`?
- Possible to print out only import block?
