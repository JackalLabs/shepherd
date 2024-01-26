install:
	go install -mod=readonly ../shepherd

build:
	go build -mod=readonly -o build/shepherd ../shepherd

PHONY: install build lint check-go-version


###############################################################################
###                                Linting                                  ###
###############################################################################

format-tools:
	go install mvdan.cc/gofumpt@v0.5.0
	gofumpt -l -w .

lint: format-tools
	golangci-lint run

format: format-tools
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs gofumpt -w -s
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs misspell -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs goimports -w -local github.com/jackalLabs/canine-chain
