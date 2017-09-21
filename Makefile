PATH:=$(GOPATH)/bin:$(PATH)

.PHONY: help
help:
	@echo 'Available commands:'
	@echo '* help					- Show this message'
	@echo '* check					- Check if required tools are installed'
	@echo '* setup					- Install required tools and dependencies'
	@echo '* hooks					- Install git hooks'
	@echo '* lint					- Lint code'
	@echo '* cover					- Coverage report'
	@echo '* test					- Run tests'

.PHONY: check
check:
ifeq ("","$(shell which go)")
	$(error go binary not in PATH)
endif
ifeq ("","$(GOPATH)")
	$(error GOPATH not configured correctly)
endif
	@test -f $(GOPATH)/bin/gometalinter || \
		echo "gometalinter binary not in $(GOPATH)/bin. run 'make setup'"
	@test -f $(GOPATH)/bin/dep || \
		echo "dep binary not in $(GOPATH)/bin. run 'make setup'"

.PHONY: setup
setup:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
	go get -u github.com/golang/dep/cmd/dep
	dep ensure

.PHONY: hooks
hooks: .git/hooks/pre-commit
.git/hooks/pre-commit:
	echo "#!/bin/sh\nmake lint" > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit

.PHONY: lint
lint: check
	@gometalinter \
		--disable-all \
		--enable=aligncheck \
		--enable=deadcode \
		--enable=gas \
		--enable=goconst \
		--enable=goimports \
		--enable=golint \
		--enable=gosimple \
		--enable=ineffassign \
		--enable=interfacer \
		--enable=misspell \
		--enable=safesql \
		--enable=staticcheck \
		--enable=structcheck \
		--enable=unparam \
		--enable=varcheck \
		--enable=vet \
		--skip vendor \
		./...

.PHONY: cover
cover: check
	@go test -cover $$(go list ./... | grep -v /vendor/)

.PHONY: test
test: check
	@go test -integration -v -race $$(go list ./... | grep -v /vendor/)
