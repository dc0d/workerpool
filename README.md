# workerpool

[![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT) [![GoDoc](https://godoc.org/github.com/dc0d/workerpool?status.svg)](https://pkg.go.dev/github.com/dc0d/workerpool/v4@v4.1.0?tab=doc) [![Go Report Card](https://goreportcard.com/badge/github.com/dc0d/workerpool)](https://goreportcard.com/report/github.com/dc0d/workerpool)


This Go Module contains an implementation of a workerpool which can get expanded &amp; shrink dynamically. Workers can get added when needed and get dismissed when no longer are needed. Of-course this workerpool can be used just as a simple one with a fixed size.

Examples can be seen inside documents.

## add as dependency modult

In the `go.mod` file of your code:

```
module github.com/user/project

go 1.14

require (
	github.com/dc0d/workerpool/v4 v4.1.0

    ...
)
```

And in the code, import it as:

```go
import "github.com/dc0d/workerpool/v4"
```

