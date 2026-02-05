## Tool Versions
ENVTEST_K8S_VERSION = 1.34.0
ENVTEST ?= $(shell go env GOPATH)/bin/setup-envtest
LOCALBIN ?= $(shell pwd)/bin

.PHONY: test
test:
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: build
build:
	go build -o $(LOCALBIN)/replik8or -trimpath -ldflags="-s -w" cmd/replik8or/main.go
