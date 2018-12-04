```
pkg: github.com/recoilme/pudge/bench
BenchmarkArtSetRand-4            2000000               958 ns/op           8.35 MB/s         161 B/op          4 allocs/op
BenchmarkArtSetOrder-4           3000000               435 ns/op          18.38 MB/s         221 B/op          7 allocs/op
BenchmarkArtSetOrderDesc-4       3000000               445 ns/op          17.98 MB/s         221 B/op          7 allocs/op
BenchmarkArtGetRand-4            5000000               407 ns/op          19.64 MB/s           0 B/op          0 allocs/op
BenchmarkHash-4                 10000000               256 ns/op          31.20 MB/s           7 B/op          0 allocs/op
BenchmarkRbtGet-4                5000000               404 ns/op          19.77 MB/s          23 B/op          1 allocs/op
BenchmarkStoreGodsBtree-4        1000000              2694 ns/op           2.97 MB/s         272 B/op          7 allocs/op
BenchmarkLoadGodsbtree-4         1000000              2064 ns/op           3.88 MB/s          47 B/op          2 allocs/op
BenchmarkSLSet-4                 2000000               829 ns/op           9.65 MB/s         243 B/op          6 allocs/op
PASS
```