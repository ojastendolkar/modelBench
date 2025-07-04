# ModelBench

**ModelBench** is a full-stack AI inference benchmarking system. It accepts input prompts from users, routes them through a Go-based orchestrator to a Python-based model-inference service (running locally via Hugging Face), and stores job metadata in PostgreSQL.

This project simulates realistic AI infrastructure by separating orchestration, model execution, and data persistence into independent services that communicate over HTTP.

---

## Architecture

```
Client
  ↓
Go Orchestrator (Gin)
  ↓
Python Inference Server (FastAPI + Transformers)
  ↓
Hugging Face model
  ↓
PostgreSQL (stores prompt, task, timestamp)
```

---

## Tech Stack

### Backend Components
- **Go (Gin)** – API server and inference orchestrator  
- **FastAPI (Python)** – Model-inference microservice  
- **PostgreSQL** – Persistent database for job data  
- **Docker Compose** – Manages Postgres and, later, service orchestration  
- **Uvicorn** – ASGI server that runs the FastAPI app  
- **Hugging Face Transformers** – Local inference with `facebook/bart-large-cnn`

### Concepts Demonstrated
- Async model serving with FastAPI and Hugging Face  
- Service decomposition: orchestrator vs. inference engine  
- Cross-language communication between Go and Python services  
- Persistent job tracking and request/response audit trail  
- REST API design and validation  
- Future extensibility for metrics, model routing, and queuing  

---

## Setup Instructions

### 1 · Start PostgreSQL

```bash
docker compose up -d
```

### 2 · Run the Go orchestrator

```bash
cd orchestrator
go run main.go
```

### 3 · Run the Python inference server

```bash
cd inference
source venv/bin/activate
uvicorn app:app --host 0.0.0.0 --port 9000
```

---

## Example Request

```bash
curl -X POST http://localhost:8000/submit   -H "Content-Type: application/json"   -d '{"prompt": "ModelBench is a benchmarking system for AI models.", "task": "summarize"}'
```

### Expected Response

```json
{
  "message": "Job stored and inference completed",
  "prompt": "ModelBench is a benchmarking system for AI models.",
  "task": "summarize",
  "output": "ModelBench is a system for benchmarking AI models."
}
```

---

## Roadmap

- [ ] Dockerize the Go and Python services  
- [ ] Expose `/metrics` in Go via `promhttp` (Prometheus integration)  
- [ ] Deploy Grafana to visualize latency and throughput  
- [ ] Persist full model output in Postgres  
- [ ] Support multiple models and dynamic task routing  
- [ ] Add queuing, batching, or rate limiting for inference  

---

## Purpose

ModelBench demonstrates:

* Distributed-system architecture for ML workloads  
* Backend service development and orchestration  
* Infrastructure-aware model serving  
* Realistic simulation of production AI workflows  
