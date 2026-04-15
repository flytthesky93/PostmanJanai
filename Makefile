.PHONY: init init-tool dev build build-win build-win-safe build-linux build-mac-amd64 build-mac-arm64 build-all ent-add

APP_NAME := PostmanJanai

init: init-tool

init-tool:
	@echo "Starting download entgo"
	go get entgo.io/ent/cmd/ent
	go generate ./ent

dev:
	wails dev

build:
	wails build

build-win:
	wails build -clean -platform windows/amd64 -o $(APP_NAME)

# Windows only: kill running exe so -clean can replace build/bin/PostmanJanai.exe
build-win-safe:
	-taskkill /IM $(APP_NAME).exe /F
	wails build -clean -platform windows/amd64 -o $(APP_NAME).exe

build-linux:
	wails build -clean -platform linux/amd64 -o $(APP_NAME)

build-mac-amd64:
	wails build -clean -platform darwin/amd64 -o $(APP_NAME)

build-mac-arm64:
	wails build -clean -platform darwin/arm64 -o $(APP_NAME)

build-all:
	wails build -clean -platform windows/amd64,linux/amd64,darwin/amd64,darwin/arm64 -o $(APP_NAME)

ent-add:
	@if [ -z "$(ENTITY)" ]; then \
		echo "Missing ENTITY variable. Use: make ent-add ENTITY=<entity-name>"; \
		exit 1; \
	fi
	go run entgo.io/ent/cmd/ent new $(ENTITY)
	go generate ./ent