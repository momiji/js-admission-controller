BIN 	= js-admissions-controller
SRC 	= $(shell find -type f -name '*.go')
DOCKER	= $(shell bash -c 'which podman &> /dev/null && echo podman || echo docker' )
RELEASE = $(shell date)

.PHONY: all
all: $(BIN)

$(BIN): $(SRC)
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
	$(DOCKER) build . -f Dockerfile.test -t js-admissions-controller:latest
	touch .make-docker

.PHONY: local
local: .make-local
.make-local: .make-docker
	$(DOCKER) tag js-admissions-controller:latest localhost:32000/js-admissions-controller:latest
	$(DOCKER) push localhost:32000/js-admissions-controller:latest
