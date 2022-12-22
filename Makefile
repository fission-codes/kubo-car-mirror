TEST_NAME = t0000-car-mirror

.PHONY: test

default: all

all: build test

clean:
	go clean ./...
	cd ../kubo && git co -- go.* plugin/*

setup-local:
	go mod edit -replace=github.com/fission-codes/go-car-mirror=../go-car-mirror
	cd ../kubo && go mod edit -replace=github.com/fission-codes/go-car-mirror=../go-car-mirror

build-core:
	go build ./...
	go build -o ./cmd/carmirror ./cmd/carmirror.go

setup-plugin:
	grep -v carmirror ../kubo/plugin/loader/preload_list > ../kubo/plugin/loader/preload_list.tmp
	echo "" >> ../kubo/plugin/loader/preload_list.tmp
	echo "carmirror github.com/fission-codes/kubo-car-mirror/plugin *" >> ../kubo/plugin/loader/preload_list.tmp
	mv ../kubo/plugin/loader/preload_list.tmp ../kubo/plugin/loader/preload_list
	$(MAKE) -C ../kubo plugin/loader/preload.go
	cd ../kubo && go mod edit -replace=github.com/fission-codes/kubo-car-mirror@v0.0.0-unpublished=../kubo-car-mirror
	cd ../kubo && go get -d github.com/fission-codes/kubo-car-mirror@v0.0.0-unpublished
	cd ../kubo && go mod tidy

build-plugin: setup-plugin
	$(MAKE) -C ../kubo build

build: build-core build-plugin

build-local: setup-local build-core build-plugin

test: test-unit sharness

test-unit:
	go test ./... -v --coverprofile=coverage.txt --covermode=atomic

sharness:
	cp test/sharness/$(TEST_NAME).sh ../kubo/test/sharness/ && cp -R test/sharness/$(TEST_NAME)-data ../kubo/test/sharness/ && cp ./cmd/carmirror ../kubo/test/bin/carmirror
	$(MAKE) -C ../kubo/test/sharness $(TEST_NAME).sh
	rm -rf ../kubo/test/sharness/$(TEST_NAME).sh ../kubo/test/sharness/$(TEST_NAME)-data ../kubo/test/bin/carmirror

sharness-v:
	cp test/sharness/$(TEST_NAME).sh ../kubo/test/sharness/ && cp -R test/sharness/$(TEST_NAME)-data ../kubo/test/sharness/ && cp ./cmd/carmirror ../kubo/test/bin/carmirror
	$(MAKE) -C ../kubo/test/sharness deps
	cd ../kubo/test/sharness && ./$(TEST_NAME).sh -v
	rm -rf ../kubo/test/sharness/$(TEST_NAME).sh ../kubo/test/sharness/$(TEST_NAME)-data ../kubo/test/bin/carmirror

sharness-no-deps:
	cp test/sharness/$(TEST_NAME).sh ../kubo/test/sharness/ && cp -R test/sharness/$(TEST_NAME)-data ../kubo/test/sharness/ && cp ./cmd/carmirror ../kubo/test/bin/carmirror
	cd ../kubo/test/sharness && ./$(TEST_NAME).sh
	rm -rf ../kubo/test/sharness/$(TEST_NAME).sh ../kubo/test/sharness/$(TEST_NAME)-data ../kubo/test/bin/carmirror

sharness-no-deps-v:
	cp test/sharness/$(TEST_NAME).sh ../kubo/test/sharness/ && cp -R test/sharness/$(TEST_NAME)-data ../kubo/test/sharness/ && cp ./cmd/carmirror ../kubo/test/bin/carmirror
	cd ../kubo/test/sharness && ./$(TEST_NAME).sh -v
	rm -rf ../kubo/test/sharness/$(TEST_NAME).sh ../kubo/test/sharness/$(TEST_NAME)-data ../kubo/test/bin/carmirror

update-deps:
	go get -u ./... & go mod tidy

update-changelog:
	conventional-changelog -p angular -i CHANGELOG.md -s

list-deps:
	go list -f '{{.Deps}}' ./... | tr "[" " " | tr "]" " " | xargs go list -f '{{if not .Standard}}{{.ImportPath}}{{end}}'
