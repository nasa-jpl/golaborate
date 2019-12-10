# ringo

[![GoDoc](https://godoc.org/github.com/brandondube/ringo?status.svg)](https://godoc.org/github.com/brandondube/ringo)

## Benchmarks

### `buf.Append()` for `CircleF64`:

```
Running tool: /usr/local/bin/go test -benchmem -run=^$ github.com/brandondube/ringo -bench ^(BenchmarkF64Append)$

goos: darwin
goarch: amd64
pkg: github.com/brandondube/ringo
BenchmarkF64Append-12    	60769256	        17.1 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/brandondube/ringo	1.474s
Success: Benchmarks passed.
```
This includes `math/rand.Float64()` overhead.

### `buf.Contiguous()` on a 10,000 element `CircleF64`:

```
Running tool: /usr/local/bin/go test -benchmem -run=^$ github.com/brandondube/ringo -bench ^(BenchmarkF64ContiguousLargeBufferFilled)$

goos: darwin
goarch: amd64
pkg: github.com/brandondube/ringo
BenchmarkF64ContiguousLargeBufferFilled-12    	  186384	      6232 ns/op	   98304 B/op	       1 allocs/op
PASS
ok  	github.com/brandondube/ringo	1.330s
Success: Benchmarks passed.
```
