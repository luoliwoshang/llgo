# LLGo baseline benchmarks

This suite is the lightweight performance gate for ordinary LLGo changes. It
uses fixed workloads and short calibrated benchmarks on Linux and macOS so it
can run on every `main` push and pull request. Branch-only series can be run
explicitly with `workflow_dispatch`, avoiding duplicate push and pull-request
jobs for the same commit. The two native jobs record normalized artifacts; a
trusted `workflow_run` publisher validates and merges both platforms into one
commit, branch, or pull-request series.

The program workloads reuse:

- `benchmark/binary_size/cprintf`: only `lib/c.Printf` (default and `-lto=full`);
- `benchmark/binary_size/println`: only the built-in `println` (default and `-lto=full`);
- `benchmark/binary_size/fmtprintf`: `fmt.Printf` (default and `-lto=full`).

For each workload, the collector performs an unmeasured warm build, then records
median build time, median process time, file size, executable-code bytes,
allocated non-executable data, and zero-filled data. On ELF, read-only constants
are included in the data bucket; on Mach-O, `__TEXT` constants are included in
the text bucket. The Go benchmark stream records five samples of selected
compiler helpers and LLGo-generated core-language operations: direct/interface
calls, defer, goroutine creation, channels, `getg`, and global access.

For pull requests, each platform job checks out the recorded base and current
commits into the same source path, then runs both suites sequentially on the same
runner. The pull request comment compares that pair, avoiding differences from
runner machines and embedded source paths. Dependency setup is shared, and Go's
build cache can be reused by unchanged packages; main pushes still run the suite
only once. Very small changes can still be scheduler, frequency, or thermal
noise and should be confirmed by repeated workflow runs. If a workflow does not
provide a paired result, the publisher falls back to the latest matching `main`
data.

The trusted publisher commits the current result history and generated site to
the `pages` branch of the configured data repository. Every LLGo repository
defaults to `<owner>/llgo-benchmark-data`:

```text
llgo/baseline/series/main/main
llgo/baseline/series/branch/<safe branch identifier>
llgo/baseline/series/pull/<number>
```

The publisher never executes code from the measured revision and pull request
jobs never receive the benchmark repository token. Pull requests receive one
updated summary comment linking to their long-term trend page. If no matching
`main` history exists yet, the pull-request report is still published and
marks every metric as `new`.

Local collection:

```sh
GOMAXPROCS=2 go build -o .benchmark/llgo ./cmd/llgo
go run ./benchmark/baseline \
  -llgo .benchmark/llgo \
  -out .benchmark/results
```

Write the selected Go benchmark output to `.benchmark/results/go.txt`:

```sh
results=.benchmark/results/go.txt
GOMAXPROCS=1 go test \
  -run '^$' \
  -bench '^(BenchmarkMergeCompilerFlags|BenchmarkMergeLinkerFlags|BenchmarkLookupPCRandom)$' \
  -benchtime=250ms -count=5 -cpu=1 \
  ./internal/clang ./internal/build/funcinfo | tee "$results"
GOMAXPROCS=1 .benchmark/llgo test \
  -run '^$' \
  -bench '^(BenchmarkRuntimeGetG|BenchmarkGlobal(Read|Write)|Benchmark(DirectCall|InterfaceCall|Defer|ChannelBuffered|ChannelHandoff))$' \
  -benchtime=250ms -count=5 \
  ./test/llgoext | tee -a "$results"
GOMAXPROCS=1 .benchmark/llgo test \
  -run '^$' -bench '^BenchmarkGoroutine$' -benchtime=100x -count=5 \
  ./test/llgoext | tee -a "$results"
```

Then validate and export the complete artifact in standard Go benchmark format:

```sh
go run ./benchmark/baseline \
  -mode export \
  -out .benchmark/results \
  -benchmark-output .benchmark/results/benchmark.txt
```
