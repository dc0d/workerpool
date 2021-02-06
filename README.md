# workerpool



[![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT) [![Go Reference](https://pkg.go.dev/badge/github.com/dc0d/workerpool/v5.svg)](https://pkg.go.dev/github.com/dc0d/workerpool/v5) [![Go Report Card](https://goreportcard.com/badge/github.com/dc0d/workerpool)](https://goreportcard.com/report/github.com/dc0d/workerpool) [![Maintainability](https://api.codeclimate.com/v1/badges/8aacea5d15dbf2295a5d/maintainability)](https://codeclimate.com/github/dc0d/workerpool/maintainability) [![Test Coverage](https://api.codeclimate.com/v1/badges/8aacea5d15dbf2295a5d/test_coverage)](https://codeclimate.com/github/dc0d/workerpool/test_coverage)


This Go Module contains an implementation of a workerpool which can get expanded &amp; shrink dynamically. Workers can get added when needed and get dismissed when no longer are needed. Of-course this workerpool can be used just as a simple one with a fixed size.

Examples can be seen inside documents.

## add as dependency modult

In the `go.mod` file of your code:

```
module github.com/user/project

go 1.15

require (
	github.com/dc0d/workerpool/v5 v5.0.1

    ...
)
```

And in the code, import it as:

```go
import "github.com/dc0d/workerpool/v5"
```

