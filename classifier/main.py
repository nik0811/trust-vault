"""
TrustVault GLiNER Classification Service

A high-performance PII/NER classification service using GLiNER zero-shot NER model.
Runs on CPU with ONNX Runtime for fast inference.
"""

import os
import time
import logging
from typing import Optional
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from classifier import GLiNERClassifier, Entity

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("trustvault-classifier")

classifier: Optional[GLiNERClassifier] = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Load model on startup, cleanup on shutdown."""
    global classifier
    
    model_name = os.getenv("MODEL_NAME", "urchade/gliner_small-v2.1")
    logger.info(f"Loading GLiNER model: {model_name}")
    
    start = time.time()
    classifier = GLiNERClassifier(model_name)
    load_time = time.time() - start
    
    logger.info(f"Model loaded in {load_time:.2f}s")
    logger.info(f"Supported entity types: {len(classifier.entity_types)}")
    
    yield
    
    logger.info("Shutting down classifier service")
    classifier = None


app = FastAPI(
    title="TrustVault Classifier",
    description="GLiNER-based PII and entity classification service",
    version="1.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


class ClassifyRequest(BaseModel):
    """Request body for classification endpoint."""
    text: str = Field(..., min_length=1, max_length=100000, description="Text to classify")
    tenant_id: str = Field(..., description="Tenant identifier")
    entity_types: Optional[list[str]] = Field(
        default=None,
        description="Specific entity types to detect. If empty, uses all supported types."
    )
    threshold: float = Field(
        default=0.5,
        ge=0.0,
        le=1.0,
        description="Minimum confidence threshold for entities"
    )


class EntityResponse(BaseModel):
    """Single detected entity."""
    text: str
    label: str
    start: int
    end: int
    confidence: float


class ClassifyResponse(BaseModel):
    """Response from classification endpoint."""
    entities: list[EntityResponse]
    processing_time_ms: float
    model: str
    text_length: int


class HealthResponse(BaseModel):
    """Health check response."""
    status: str
    model_loaded: bool
    model_name: str
    supported_entities: int


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    return HealthResponse(
        status="healthy" if classifier is not None else "unhealthy",
        model_loaded=classifier is not None,
        model_name=classifier.model_name if classifier else "not loaded",
        supported_entities=len(classifier.entity_types) if classifier else 0,
    )


@app.get("/ready")
async def readiness_check():
    """Readiness probe for Kubernetes/Docker."""
    if classifier is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    return {"status": "ready"}


@app.get("/entities")
async def list_entities():
    """List all supported entity types."""
    if classifier is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    return {
        "entity_types": classifier.entity_types,
        "count": len(classifier.entity_types),
    }


@app.post("/classify", response_model=ClassifyResponse)
async def classify_text(request: ClassifyRequest):
    """
    Classify text for PII and named entities.
    
    Uses GLiNER zero-shot NER model to detect entities without requiring
    training data for each entity type.
    """
    if classifier is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    start_time = time.time()
    
    entity_types = request.entity_types
    if not entity_types:
        entity_types = classifier.entity_types
    
    entities = classifier.predict(
        text=request.text,
        labels=entity_types,
        threshold=request.threshold,
    )
    
    processing_time = (time.time() - start_time) * 1000
    
    response_entities = [
        EntityResponse(
            text=e.text,
            label=e.label,
            start=e.start,
            end=e.end,
            confidence=e.confidence,
        )
        for e in entities
    ]
    
    logger.info(
        f"Classified {len(request.text)} chars, found {len(entities)} entities "
        f"in {processing_time:.1f}ms (tenant={request.tenant_id})"
    )
    
    return ClassifyResponse(
        entities=response_entities,
        processing_time_ms=round(processing_time, 2),
        model=classifier.model_name,
        text_length=len(request.text),
    )


@app.post("/classify/batch")
async def classify_batch(requests: list[ClassifyRequest]):
    """
    Batch classification for multiple texts.
    
    More efficient than individual requests for processing multiple documents.
    """
    if classifier is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    results = []
    total_start = time.time()
    
    for req in requests:
        start_time = time.time()
        
        entity_types = req.entity_types or classifier.entity_types
        entities = classifier.predict(
            text=req.text,
            labels=entity_types,
            threshold=req.threshold,
        )
        
        processing_time = (time.time() - start_time) * 1000
        
        results.append({
            "tenant_id": req.tenant_id,
            "entities": [
                {
                    "text": e.text,
                    "label": e.label,
                    "start": e.start,
                    "end": e.end,
                    "confidence": e.confidence,
                }
                for e in entities
            ],
            "processing_time_ms": round(processing_time, 2),
            "text_length": len(req.text),
        })
    
    total_time = (time.time() - total_start) * 1000
    
    return {
        "results": results,
        "total_processing_time_ms": round(total_time, 2),
        "batch_size": len(requests),
    }


if __name__ == "__main__":
    port = int(os.getenv("PORT", "8085"))
    host = os.getenv("HOST", "0.0.0.0")
    
    logger.info(f"Starting TrustVault Classifier on {host}:{port}")
    
    uvicorn.run(
        "main:app",
        host=host,
        port=port,
        reload=os.getenv("ENV", "production") == "development",
        workers=1,
    )
