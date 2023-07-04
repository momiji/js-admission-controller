BIN             = js-admissions-controller
SRC             = $(shell find -type f -name '*.go')
DOCKER         ?= $(shell bash -c 'which podman &> /dev/null && echo podman || echo docker' )
RELEASE         = $(shell date)
REGISTRY_PORT  ?= 32000
BUILD_OPTS      =

ifeq ($(DOCKER),podman)
BUILD_OPTS      = --tls-verify=false
endif

.PHONY: all
all: $(BIN)

$(BIN): $(SRC) go.mod go.sum
	go build -o $(BIN) -ldflags="-w -s -X 'main.Version=$(RELEASE)'"

.PHONY: run
run: $(BIN)
	KUBECONFIG=~/.kube/config ./$(BIN) --tlsCert ./tests/certs/tls.crt --tlsKey ./tests/certs/tls.key

.PHONY: clean
clean:
	rm -f $(BIN) .make-*

.PHONY: docker
docker: .make-docker
.make-docker: Dockerfile $(BIN)
	sudo $(DOCKER) build . -f Dockerfile.test -t js-admissions-controller:latest
	touch .make-docker

.PHONY: local
local: .make-local
.make-local: .make-docker
	sudo $(DOCKER) tag js-admissions-controller:latest localhost:$(REGISTRY_PORT)/js-admissions-controller:latest
	sudo $(DOCKER) push localhost:$(REGISTRY_PORT)/js-admissions-controller:latest $(BUILD_OPTS)
