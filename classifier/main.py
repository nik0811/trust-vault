"""TrustVault GLiNER Classification Service - Minimal Python sidecar"""
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Optional
import time
import os

app = FastAPI(title="TrustVault GLiNER Classifier", version="1.0.0")

# Global model instance
model = None
model_name = "gliner-pii"

class ClassifyRequest(BaseModel):
    text: str
    entity_types: Optional[list[str]] = None
    threshold: float = 0.5

class Entity(BaseModel):
    type: str
    value: str
    start: int
    end: int
    confidence: float

class ClassifyResponse(BaseModel):
    entities: list[Entity]
    processing_ms: int
    model_used: str

# Default PII entity types
DEFAULT_ENTITIES = [
    "EMAIL", "PHONE", "SSN", "CREDIT_CARD", "IBAN", "IP_ADDRESS",
    "PERSON", "ADDRESS", "DATE_OF_BIRTH", "PASSPORT", "DRIVER_LICENSE",
    "BANK_ACCOUNT", "TAX_ID", "MEDICAL_RECORD", "HEALTH_INSURANCE_ID"
]

def test_model(m) -> bool:
    """Quick test to verify model works"""
    try:
        result = m.predict_entities("test@email.com", ["email"], threshold=0.1)
        return len(result) > 0
    except:
        return False

def load_model():
    """Load GLiNER model - tries local first, falls back to HuggingFace"""
    global model, model_name
    from gliner import GLiNER
    
    model_dir = os.getenv("MODEL_DIR", "/models")
    hf_model = os.getenv("GLINER_MODEL", "urchade/gliner_small-v2.1")
    
    # Try local ONNX model first
    onnx_path = os.path.join(model_dir, "gliner-pii-edge-int8.onnx")
    if os.path.exists(onnx_path):
        try:
            m = GLiNER.from_pretrained(model_dir, load_onnx_model=True, onnx_model_file="gliner-pii-edge-int8.onnx")
            if test_model(m):
                model = m
                model_name = "gliner-pii-onnx"
                print(f"Loaded ONNX model from {onnx_path}")
                return
            print("ONNX model loaded but failed test, trying alternatives...")
        except Exception as e:
            print(f"ONNX load failed: {e}")
    
    # Try local PyTorch model
    pytorch_path = os.path.join(model_dir, "pytorch_model.bin")
    if os.path.exists(pytorch_path):
        try:
            m = GLiNER.from_pretrained(model_dir)
            if test_model(m):
                model = m
                model_name = "gliner-pii-pytorch"
                print(f"Loaded PyTorch model from {model_dir}")
                return
            print("PyTorch model loaded but failed test, trying HuggingFace...")
        except Exception as e:
            print(f"PyTorch load failed: {e}")
    
    # Load from HuggingFace
    try:
        print(f"Loading model from HuggingFace: {hf_model}")
        model = GLiNER.from_pretrained(hf_model)
        model_name = hf_model.split("/")[-1]
        print(f"Loaded model from HuggingFace: {hf_model}")
    except Exception as e:
        print(f"All model loading failed: {e}")
        model = None

@app.on_event("startup")
async def startup():
    load_model()

@app.get("/health")
async def health():
    return {"status": "ok" if model else "degraded", "model": model_name}

@app.get("/health/ready")
async def ready():
    if model is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    return {"status": "ready", "model": model_name}

@app.post("/classify", response_model=ClassifyResponse)
async def classify(req: ClassifyRequest):
    if model is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    if not req.text:
        raise HTTPException(status_code=400, detail="text field is required")
    
    start = time.time()
    
    # Use provided entity types or defaults
    labels = req.entity_types if req.entity_types else DEFAULT_ENTITIES
    
    # Run GLiNER prediction
    try:
        predictions = model.predict_entities(req.text, labels, threshold=req.threshold)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Classification failed: {e}")
    
    # Convert to response format
    entities = [
        Entity(
            type=p["label"],
            value=p["text"],
            start=p["start"],
            end=p["end"],
            confidence=round(p["score"], 4)
        )
        for p in predictions
    ]
    
    processing_ms = int((time.time() - start) * 1000)
    
    return ClassifyResponse(
        entities=entities,
        processing_ms=processing_ms,
        model_used=model_name
    )

@app.post("/classify/batch")
async def classify_batch(items: list[ClassifyRequest]):
    """Batch classification endpoint"""
    if model is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    start = time.time()
    results = []
    
    for req in items:
        labels = req.entity_types if req.entity_types else DEFAULT_ENTITIES
        try:
            predictions = model.predict_entities(req.text, labels, threshold=req.threshold)
            entities = [
                Entity(
                    type=p["label"],
                    value=p["text"],
                    start=p["start"],
                    end=p["end"],
                    confidence=round(p["score"], 4)
                )
                for p in predictions
            ]
            results.append({"entities": entities, "char_count": len(req.text)})
        except Exception as e:
            results.append({"entities": [], "error": str(e)})
    
    return {
        "results": results,
        "total_ms": int((time.time() - start) * 1000),
        "item_count": len(items)
    }

@app.get("/info")
async def info():
    return {
        "version": "1.0.0",
        "model": model_name,
        "ready": model is not None,
        "default_entities": DEFAULT_ENTITIES
    }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8085)
