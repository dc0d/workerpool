# workerpool

[![License MIT](https://img.shields.io/badge/License-MIT-blue.svg)](http://opensource.org/licenses/MIT) [![GoDoc](https://godoc.org/github.com/dc0d/workerpool?status.svg)](http://godoc.org/github.com/dc0d/workerpool) [![Go Report Card](https://goreportcard.com/badge/github.com/dc0d/workerpool)](https://goreportcard.com/report/github.com/dc0d/workerpool) [![Build Status](https://travis-ci.org/dc0d/workerpool.svg?branch=master)](http://travis-ci.org/dc0d/workerpool) [![codecov](https://codecov.io/gh/dc0d/workerpool/branch/master/graph/badge.svg)](https://codecov.io/gh/dc0d/workerpool)

Get the package using:

```
$ go get -u -v -tags v3 github.com/dc0d/workerpool
```

* No longer uses the `context.Context` pattern,
* Channels removed from the API and replaced by simple function calls,

Previous version:

```
$ go get -u -v -tags v2 github.com/dc0d/workerpool
```

This is an implementation of a workerpool which can get expanded &amp; shrink dynamically. Workers can get added when needed and get dismissed when no longer are needed. Of-course this workerpool can be used just as a simple one with a fixed size.

Examples can be seen inside documents.