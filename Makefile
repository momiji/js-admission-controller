BIN 	= js-admissions-controller
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
	$(DOCKER) build . -f Dockerfile.test -t js-admissions-controller:latest

.PHONY: local
local:
	$(DOCKER) tag js-admissions-controller:latest localhost:32000/js-admissions-controller:latest
	$(DOCKER) push localhost:32000/js-admissions-controller:latest
