BIN 	= js-admission-controller
SRC 	= $(shell find -type f -name '*.go')
DOCKER	= $(shell bash -c 'which podman &> /dev/null && echo podman || echo docker' )

.PHONY: all
all: $(BIN)

$(BIN): $(SRC)
	go build -o $(BIN)

.PHONY: run
run: $(BIN)
	KUBECONFIG=~/.kube/config ./$(BIN) --tlsCert ./kubernetes/tests/certs/tls.crt --tlsKey ./kubernetes/tests/certs/tls.key

.PHONY: clean
clean:
	rm $(BIN)

.PHONY: docker
docker:
	$(DOCKER) build . -f Dockerfile.test -t js-admission-controller:latest

.PHONY: local
local:
	$(DOCKER) tag js-admission-controller:latest localhost:32000/js-admission-controller:latest
	$(DOCKER) push localhost:32000/js-admission-controller:latest
