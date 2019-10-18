.PHONY: all

PKG_LIST:=$(shell go list ./... | grep -v /vendor/)
EXECUTABLE:=lenses-cli
OUTPUT:=bin
LDFLAGS:= -ldflags "-s -w -X main.buildVersion=${VERSION} -X main.buildRevision=${REVISION} -X main.buildTime=$(shell date +%s)"

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: dep ## Build for development only
	 go build -o ./${OUTPUT}/${EXECUTABLE} ./cmd/${EXECUTABLE}

build-linux: dep ## Build binary for linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o ${OUTPUT}/${EXECUTABLE}-linux-amd64 ./cmd/${EXECUTABLE}

build-docker: ## Builds Docker with linux lenses-cli
	docker build -t landoop/lenses-cli:${VERSION} .
	docker build -t lensesio/lenses-cli:${VERSION} .

dep: ## Ensure dependencies
	go mod verify

clean: dep ## Clean
	go clean
	rm -r bin/
	rm -f cover.out

cross-build: dep ## Build the app for multiple os/arch
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o ${OUTPUT}/${EXECUTABLE}-darwin-amd64 ./cmd/${EXECUTABLE}
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o ${OUTPUT}/${EXECUTABLE}-linux-amd64 ./cmd/${EXECUTABLE}
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build ${LDFLAGS} -o ${OUTPUT}/${EXECUTABLE}-linux-386 ./cmd/${EXECUTABLE}
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ${LDFLAGS} -o ${OUTPUT}/${EXECUTABLE}-linux-arm64 ./cmd/${EXECUTABLE}
	GOOS=linux GOARCH=arm CGO_ENABLED=0 go build ${LDFLAGS} -o ${OUTPUT}/${EXECUTABLE}-linux-arm ./cmd/${EXECUTABLE}
	GOOS=windows GOARCH=386 CGO_ENABLED=0 go build ${LDFLAGS} -o ${OUTPUT}/${EXECUTABLE}-windows-386.exe ./cmd/${EXECUTABLE}
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o ${OUTPUT}/${EXECUTABLE}-windows-amd64.exe ./cmd/${EXECUTABLE}

lint: ## Linting the codebase
	golint -set_exit_status ${PKG_LIST}

publish: ## Publish lenses CLI as docker
	bash -c "./publish-docker"

race: dep ## Run data race detector
	go test -race -short ${PKG_LIST}

setup: ## Get all the necessary dependencies 
	go get -u github.com/golang/dep/cmd/dep
	go get -u golang.org/x/lint/golint

test: dep ## Run tests
	go test -coverprofile=cover.out ./...
	go tool cover -func=cover.out
