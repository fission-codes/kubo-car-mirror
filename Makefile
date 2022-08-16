# GOFILES = $(shell find . -name '*.go' -not -path './vendor/*')

default: test

clean:
	go clean ./...

build:
	go build ./...

test:
	go test ./... -v --coverprofile=coverage.txt --covermode=atomic

sharness:
	cp test/sharness/t0000-car-mirror.sh ../kubo/test/sharness/ && cp -R test/sharness/t0000-car-mirror-data ../kubo/test/sharness/
	cd ../kubo/test/sharness && ./t0000-car-mirror.sh -v
	rm -rf ../kubo/test/sharness/t0000-car-mirror.sh ../kubo/test/sharness/t0000-car-mirror-data

update-deps:
	go get -u ./... & go mod tidy

update-changelog:
	conventional-changelog -p angular -i CHANGELOG.md -s

list-deps:
	go list -f '{{.Deps}}' ./... | tr "[" " " | tr "]" " " | xargs go list -f '{{if not .Standard}}{{.ImportPath}}{{end}}'
