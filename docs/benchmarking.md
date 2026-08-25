# Benchmarking protocol

Do not put unmeasured numbers on a resume. Run benchmarks on an otherwise idle
machine, capture five trials, and report the median throughput plus p50 and p95
latency. Include the CPU model, memory, operating system, dataset size, process
count, and commit SHA.

## Search

1. Ingest at least 500,000 documents from a fixed Wikipedia dump.
2. Warm the page cache and result cache with one complete query pass.
3. Run a 70/20/10 mix of common, medium, and rare queries for five minutes.
4. Repeat while terminating one replica per shard.

## Key-value store

Use uniform and Zipfian key distributions with 50/50 and 95/5 read/write
mixes. Test one, three, and five nodes. Record leader failover time and verify
the linearizability history with the included checker interface.

## Event analytics

Increase producer rate until processing lag grows for three consecutive
minutes. Record the last stable event rate, duplicate rate, late-event rate,
consumer count, partition count, and p95 end-to-end lag.

## Video

Use fixed one-minute 1080p inputs. Record completion latency, worker CPU,
retry counts, and success rate while terminating 10 percent of active workers.

## Time series

Ingest a fixed set of 1,000 labeled series at constant and burst rates. Record
samples/second, bytes/sample after flush, WAL replay time, and p50/p95 range
query latency for one-hour, one-day, and seven-day windows.
