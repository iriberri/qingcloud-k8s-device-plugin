.PHONY: build doc fmt lint run test vendor_clean vendor_get vendor_update vet
RELEASE_TAG?=$(shell git describe --abbrev=0 --tags)
GIT_VERSION?= $(shell git --no-pager describe --tags --always --dirty)
GIT_STATE?= $(shell result=$(git diff-index HEAD --) && test -n $result && echo dirty )
default: build

build: vet
	go build -v -ldflags "-X github.com/gravitational/version.version=$(RELEASE_TAG) -X github.com/gravitational/version.gitCommit=$(GIT_VERSION) -X github.com/gravitational/version.gitTreeState=$(GIT_STATE)" -o ./out/qingcloud-gpu ./cmd/qingcloud-gpu
	tar czvf out/qingcloud_gpu_linux_x86_64.tar.gz -C out qingcloud-gpu

lint:
	golint ./

run: build
	./out/qingcloud-gpu

vendor_clean:
	rm -dRf ./vendor

# We have to set GOPATH to just the _vendor
# directory to ensure that `go get` doesn't
# update packages in our primary GOPATH instead.
# This will happen if you already have the package
# installed in GOPATH since `go get` will use
# that existing location as the destination.
vendor_get: vendor_clean
	dep ensure

vendor_update: vendor_get
	dep ensure -update

vet:
	go vet ./...

clean:
	rm -rf out/
