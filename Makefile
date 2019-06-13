GO := go

all: clean format build

clean:
	@rm -rf server

format:
	@echo ">> formatting code"
	@$(GO) fmt main/Main.go

build:
	@echo ">> building binaries"
	@GOOS=linux GOARCH=amd64 $(GO) build -o server main/Main.go
