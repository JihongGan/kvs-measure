# EECS 491 KVS Benchmarks
Measure the throughput and latency of your KV store.

# Usage
Paste `kvpaxos_perf_test.go` into `<working dir>/kvpaxos`, and run the test cases inside.
Build more benchmarks with different combinations of value size, number of servers, number of clients.

## Sample output
```
Benchmarking with value size 512 bytes, 4 servers, 16 clients...
Running read-only workload...
Throughput: 1058.10 ops/s
Average latency: 14.54 ms
Running half-read-half-write workload...
Throughput: 1023.97 ops/s
Average latency: 15.03 ms
```
