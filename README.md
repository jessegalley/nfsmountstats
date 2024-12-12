# nfsmountstats

A package for parsing NFS performance stats on linux. 

Parses counters from `/proc/self/mountstats` into structs for easy processing.

### examples:

Get basic info including read/write operations and bytes:
```go run examples/basic/basic.go```

Get cache info like page cache hitrate and attribute cache revalidates:
```go run examples/cache/cache.go```
