
# Cassandra Coalescing POC

A small Go proof of concept for experimenting with **Cassandra, request coalescing, connection pooling, and high-concurrency workloads**.

## Architecture

```text
Load Test
    │
    ▼
 Go API
    │
    │ singleflight
    ▼
Cassandra Cluster
 ┌──────┬──────┐
 ▼      ▼      ▼
Node 1 Node 2 Node 3
````

## Run

Start Cassandra:

```bash
docker compose up -d
```

Check the cluster:

```bash
docker exec cassandra-1 nodetool status
```

Initialize the database:

```bash
docker exec cassandra-1 cqlsh -f /init.cql
```

Start the API:

```bash
docker compose up -d --build api
```

Run the load test:

```bash
go run ./loadtest
```

Runnign with custom request and workers:

```bash
TOTAL_REQUESTS=1000000 WORKERS=1000 go run ./loadtest
```
