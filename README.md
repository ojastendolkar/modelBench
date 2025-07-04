# ModelBench

ModelBench is a backend system for benchmarking the inference performance of multiple language models (e.g., BART, DistilBERT, Mistral).

## Features

- Submit prompts and tasks (e.g. summarize, sentiment)
- Run inference across different models
- Measure latency and token count
- Store results in PostgreSQL
- Optional: expose metrics via Prometheus and visualize in Grafana

## Tech Stack

- Go (orchestrator)
- Python (inference workers)
- PostgreSQL
- Docker Compose
- Optional: Prometheus + Grafana
