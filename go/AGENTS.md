Go implementation of the shittycannon autocannon subset.

The binary is `cmd/` (`package main`). Library packages are `runoptions`, `cannon`, `report`, and `stats`.

Tests live next to the code as `go test ./...`. Format with `gofmt`. Lint with
`make lint` (`golangci-lint` v2, config `.golangci.yml`). That includes
`modernize`, the same analyzer suite as gopls (e.g. `WaitGroup.Go`). After a
task, run `make check` (test, lint, gofmt -l). Start the CLI with
`go run ./cmd -- <args>`.
