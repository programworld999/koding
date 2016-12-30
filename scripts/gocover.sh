#!/usr/bin/env bash
filepath=$1

go list -f '{{if len .TestGoFiles}}"go test -v -race -coverprofile={{.Dir}}/{{.Name}}.coverprofile {{.ImportPath}}"{{end}}' $filepath | xargs -L 1 sh -c
