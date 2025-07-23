from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from transformers import pipeline
import time

app = FastAPI()

# Preload multiple models (for simplicity we use the same pipeline with different aliases)
MODEL_REGISTRY = {
    "bart": pipeline("summarization", model="facebook/bart-large-cnn"),
    "t5": pipeline("summarization", model="t5-small")
    # You can add more models here
}

class InferenceRequest(BaseModel):
    prompt: str
    task: str  # optional, keeping for compatibility
    model_id: str  # NEW: defines which model to use

@app.post("/infer")
def infer(req: InferenceRequest):
    if req.model_id not in MODEL_REGISTRY:
        raise HTTPException(status_code=400, detail=f"Model {req.model_id} not supported")

    model = MODEL_REGISTRY[req.model_id]

    start = time.time()
    output = model(req.prompt, max_length=60, min_length=20, do_sample=False)[0]["summary_text"]
    latency = time.time() - start

    return {
        "output": output,
        "latency": latency,
        "model_used": req.model_id
    }
