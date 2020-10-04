[![Go Report Card](https://goreportcard.com/badge/github.com/quasilyte/minformat)](https://goreportcard.com/report/github.com/quasilyte/minformat)
[![GoDoc](https://godoc.org/github.com/quasilyte/minformat/go/minformat?status.svg)](https://godoc.org/github.com/quasilyte/minformat/go/minformat)

# `go/minformat`

This package formats the Go source code in a way so it becomes more compact.
It can be considered to be a minifier, although it doesn't make irreversible transformations by default (well, it does remove all comments).

The result can become readable again after running `go/format`, making this pipeline possible:

1. Minify the code before transferring it over a network
2. Send the (potentially further compressed) minified source code
3. On the receiving side, run `gofmt` to get the canonical formatting

For (3) I would recommend using [gofumpt](https://github.com/mvdan/gofumpt).

## Example

Suppose that we have this `hello.go` file:

```go
package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello, playground")
}
```

It will be formatted into this:

```go
package main;import("fmt");func main(){fmt.Println("Hello, playground")}
```

Depending on the file, it usually cuts 10-50% of the file size.
