# The name of the executable (default is current directory name)
TARGET := $(shell echo $${PWD\#\#*/})
.DEFAULT_GOAL: $(TARGET)

# These will be provided to the target
VERSION := 0.1.0
BUILD := `git rev-parse --short HEAD`
PLATFORMS := linux/386 linux/amd64 linux/arm linux/arm64 darwin/amd64
# GH
GH_USER := r4scal
GH_REPO := zyxel-gs1920-exporter

# Use linker flags to provide version/build settings to the target
LDFLAGS=-X=main.appVersion=$(VERSION)
LDFLAGS+=-X=main.shortSha=$(BUILD)

# go source files, ignore vendor directory
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

all: fmt misspell clean build

temp = $(subst /, ,$@)
os = $(word 1, $(temp))
arch = $(word 2, $(temp))

build:
	go build -ldflags "$(LDFLAGS)" .

deploy: utils
	CGO_ENABLED=0 gox -os="linux" -arch="amd64 arm arm64 386" -parallel=4 -ldflags "$(LDFLAGS)" -output "dist/$(GH_REPO)_{{.OS}}_{{.Arch}}"
	ghr -u $(GH_USER) -r $(GH_REPO)  -replace v$(VERSION) dist/

clean:
	@rm -f bin/$(TARGET)

deps:
	@echo "==> Install deps ..."; \
	GO111MODULE=$(GO111MODULE) go mod tidy; \
	GO111MODULE=$(GO111MODULE) go get -u -d;

fmt:
	@echo "==> Formatting ..."; \
	gofmt -l -w $(SRC)

gocyclo:
	@echo "==> Gocyclo ..."; \
	gocyclo -top 10 $(SRC)

ineffassign:
	@echo "==> Ineffassign ..."; \
	ineffassign $(SRC)

misspell:
	@echo "==> Misspell ..."; \
	misspell -w $(SRC)

lint:
	@for p in $(SRC); do \
		echo "==> Lint $$p ..."; \
		golint ./$$p; \
	done

simplify:
	@echo "==> Simplify ..."; \
	gofmt -s -l -w $(SRC)

vet:
	@for p in $(PACKAGE_LIST); do \
		echo "==> Vet $$p ..."; \
		go vet ./$$p; \
	done

utils:
	go get github.com/mitchellh/gox
	go get github.com/tcnksm/ghr