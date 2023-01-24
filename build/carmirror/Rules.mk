include mk/header.mk

KUBO_CAR_MIRROR_GIT_VERSION ?= $(shell git ls-remote --refs --heads https://github.com/fission-codes/kubo-car-mirror.git main | awk '{print $$1}')

build-carmirror:
	grep -v carmirror plugin/loader/preload_list > plugin/loader/preload_list.tmp
	echo "" >> plugin/loader/preload_list.tmp
	echo "carmirror github.com/fission-codes/kubo-car-mirror/plugin *" >> plugin/loader/preload_list.tmp
	mv plugin/loader/preload_list.tmp plugin/loader/preload_list
	$(MAKE) plugin/loader/preload.go
	go get -d github.com/fission-codes/kubo-car-mirror@$(KUBO_CAR_MIRROR_GIT_VERSION)
	go mod tidy
	go get github.com/fission-codes/kubo-car-mirror/cmd/carmirror@$(KUBO_CAR_MIRROR_GIT_VERSION)
	GOBIN=$(PWD)/carmirror/cmd/carmirror go install github.com/fission-codes/kubo-car-mirror/cmd/carmirror@$(KUBO_CAR_MIRROR_GIT_VERSION)

include mk/footer.mk
