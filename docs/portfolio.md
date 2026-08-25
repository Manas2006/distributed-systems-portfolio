# Resume evidence checklist

Replace every placeholder only with a result reproduced from the benchmark
protocol. Link the repository and a short architecture note from the resume.

## Atlas Search

- Built a distributed search engine in Go that indexed **N** Wikipedia
  documents with compressed inverted indexes and BM25 ranking, serving **Q**
  queries/second at **L ms** p95 latency.
- Implemented rendezvous-hash sharding, replica failover, and write-ahead-log
  recovery, restoring acknowledged updates in **R seconds** during injected
  process failures.

## Quorum KV

- Implemented a replicated key-value store in Go with Raft leader election,
  quorum writes, write-ahead logging, and snapshot recovery across **N** nodes.
- Sustained **Q operations/second** while preserving committed reads and writes
  through leader termination and delayed-message tests.

## Pulse Analytics

- Built a Java/Kafka event pipeline processing **E events/second** with
  event-time windows, idempotent consumers, and explicit offset checkpoints.
- Maintained **L-second** p95 processing lag during a **S x** traffic spike
  using bounded queues, partition rebalancing, and backpressure.

## StreamForge

- Built a Go/FFmpeg video platform that converted concurrent uploads into
  multi-bitrate HLS streams using resumable object writes and lease-based jobs.
- Achieved **P percent** job completion during worker termination through
  idempotent outputs, heartbeats, exponential retries, and dead-letter replay.

## Chronos TSDB

- Built a monitoring database in Go that compressed **N** time-series samples
  into immutable delta-of-delta/XOR blocks and served **Q** range queries/second.
- Implemented fsynced WAL ingestion, restart recovery, labeled series lookup,
  and retention deletion while reducing stored bytes per sample to **B**.
