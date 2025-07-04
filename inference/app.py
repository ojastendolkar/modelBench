from fastapi import FastAPI
from pydantic import BaseModel
from transformers import pipeline

# Initialize FastAPI app
app = FastAPI()

# Load summarization model once at startup
summarizer = pipeline("summarization", model="facebook/bart-large-cnn")

# Define expected request schema
class InferenceRequest(BaseModel):
    prompt: str
    task: str

# Define inference endpoint
@app.post("/infer")
async def infer(request: InferenceRequest):
    # For now, only support summarization
    if request.task != "summarize":
        return {"error": f"Unsupported task: {request.task}"}
    
    # Run model inference
    result = summarizer(
        request.prompt,
        max_length=100,
        min_length=10,
        do_sample=False
    )

    # Return summarized output
    return {"output": result[0]["summary_text"]}
