# Memory Profiling Sidecar

This guide explains how to use Go's pprof to diagnose memory issues in sidecar.

## Enabling Profiling

Set the `SIDECAR_PPROF` environment variable to enable the pprof HTTP server:

```bash
# Default port 6060
SIDECAR_PPROF=1 sidecar

# Custom port
SIDECAR_PPROF=6061 sidecar
```

You'll see a message on startup: `pprof enabled on http://localhost:6060/debug/pprof/`

## Disabling Profiling

Simply don't set the environment variable:

```bash
sidecar  # No pprof server, normal operation
```

## Capturing Profiles

### Heap Profile (Current Allocations)

Shows what memory is currently allocated:

```bash
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof -top heap.prof

# Interactive mode
go tool pprof heap.prof
# Commands: top20, list <function>, web (opens browser)
```

### Allocs Profile (All Allocations)

Shows all allocations since program start, including freed memory:

```bash
curl http://localhost:6060/debug/pprof/allocs > allocs.prof
go tool pprof -top allocs.prof
```

### Goroutine Profile

Check for goroutine leaks:

```bash
# Count (first line shows total)
curl -s http://localhost:6060/debug/pprof/goroutine?debug=1 | head -1

# Full stacks (text format)
curl http://localhost:6060/debug/pprof/goroutine?debug=2 > goroutines.txt

# Look for stuck goroutines
grep -A5 'runtime.chanrecv' goroutines.txt
```

### Memory Stats

Quick view of memory metrics:

```bash
curl http://localhost:6060/debug/pprof/heap?debug=1 | head -30
# Shows: HeapAlloc, HeapInuse, HeapSys, etc.
```

## Comparing Snapshots Over Time

This is the most useful technique for finding leaks:

```bash
# Take baseline
curl http://localhost:6060/debug/pprof/heap > heap1.prof

# Wait (e.g., 1 hour, or overnight)
curl http://localhost:6060/debug/pprof/heap > heap2.prof

# Compare (shows what grew)
go tool pprof -base heap1.prof heap2.prof
# Then: top20
```

## Continuous Monitoring

Use the included monitoring script for ongoing observation:

```bash
./scripts/mem-monitor.sh         # Default: port 6060, 60s interval
./scripts/mem-monitor.sh 6061 30 # Custom port and interval
```

Output is CSV format: `time,heap_alloc_bytes,heap_inuse_bytes,goroutines,rss_mb`

Results are logged to `mem-YYYYMMDD-HHMMSS.log`.

## What to Look For

### Goroutine Leaks

- Count should stabilize after startup (typically 20-50 for sidecar)
- Steady growth = leak
- Search goroutine dump for: `runtime.chanrecv1`, `runtime.chansend1`, `time.Sleep`

### Heap Growth

- `HeapAlloc` should stabilize after loading sessions
- Consistent upward trend over hours = leak
- Spikes that don't recede = retained memory

### Common Leak Signatures

- `bufio.Scanner` - buffer not returned to pool
- `json.Unmarshal` - large objects retained
- `append` in loops - slice capacity growing
- Channel operations - blocked senders/receivers

## Overnight Test Procedure

1. Start with profiling: `SIDECAR_PPROF=1 sidecar`
2. Take baseline: `curl .../heap > baseline.prof`
3. Note goroutine count
4. Leave overnight (use `caffeinate` on macOS to prevent sleep)
5. Morning: capture new profiles
6. Compare: `go tool pprof -base baseline.prof morning.prof`
7. Check goroutines for stuck ones

## Web UI

pprof also provides a web interface:

```bash
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

This opens a browser with flame graphs, call graphs, and more.

## Notes

- pprof adds ~1-2MB memory overhead
- The HTTP server runs in a separate goroutine
- Profile endpoints are only accessible from localhost
- No performance impact on the TUI when profiling is enabled
