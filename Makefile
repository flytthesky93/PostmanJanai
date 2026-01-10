init:
	init-tool

init-tool:
	echo "Starting download entgo"
	go get entgo.io/ent/cmd/ent
	go generate ./ent

dev:
	wails dev
build:
	wails build

ent-add:
	@if ["$(ENTITY)" = ""]; then \
  		echo "Missing ENTITY variable in ent-add command. your command should be: make ent-add ENTITY=<your-entity-name>"; \
  	else \
		go run entgo.io/ent/cmd/ent new $(ENTITY); \
	fi
	go generate ./ent