# nxretry

[![Godoc Reference][pkg.go.dev_img]][pkg.go.dev]
[![Pipeline Status][pipeline_img]][pipeline]

Context-aware retrier with an optional and customizable backoff.

[pkg.go.dev]: https://pkg.go.dev/github.com/matthewpi/nxretry
[pkg.go.dev_img]: https://img.shields.io/badge/%E2%80%8B-reference-007d9c?logo=go&logoColor=white&style=flat-square
[pipeline]: https://github.com/matthewpi/nxretry/actions/workflows/ci.yaml
[pipeline_img]: https://img.shields.io/github/actions/workflow/status/matthewpi/nxretry/ci.yaml?style=flat-square&label=tests

## Installation

```bash
go get github.com/matthewpi/nxretry
```

## Usage

```go
r := nxretry.New(
	nxretry.MaxAttempts(3),
	nxretry.Exponential{
		Factor: 2,
		Min:    1 * time.Second,
		Max:    5 * time.Second,
	},
)

// Run code with the ability to retry, optionally using the provided context.
for ctx := range r.Next(context.Background()) {
	_ = ctx

	// Do something.
	//
	// `break` on success (or if you don't want to retry anymore) and `continue` on failure.
}
```

See [`example_test.go`](./example_test.go) or the [Godoc reference](https://pkg.go.dev/github.com/matthewpi/nxretry) for more examples

## Licensing

All code in this repository is licensed under the [MIT license](./LICENSE).

This package includes **ZERO** external dependencies, including any `golang.org/x` packages.
