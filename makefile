.PHONY: generr test fmt vet tidy check

generr:
	glitch gen -y errors/*.yaml -p gerrors --out internal/gerrors

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

check: generr fmt vet test
