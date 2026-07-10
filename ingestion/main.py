from fastapi import FastAPI, HTTPException, BackgroundTasks
from pydantic import BaseModel
from typing import Optional, Dict, Any
import subprocess
import json
import os
import tempfile
import yaml

app = FastAPI(title="TrustVault Ingestion Sidecar", version="1.0.0")

class IngestionRequest(BaseModel):
    datasource_id: str
    tenant_id: str
    type: str
    config: Dict[str, Any]
    callback_url: Optional[str] = None

class IngestionStatus(BaseModel):
    job_id: str
    status: str
    message: Optional[str] = None
    datasets_discovered: int = 0

# Recipe templates for different source types
RECIPE_TEMPLATES = {
    "postgres": """
source:
  type: postgres
  config:
    host_port: "{host}:{port}"
    database: "{database}"
    username: "{username}"
    password: "{password}"
    include_tables: true
    include_views: true
    profiling:
      enabled: true
sink:
  type: datahub-rest
  config:
    server: "{datahub_url}"
""",
    "mysql": """
source:
  type: mysql
  config:
    host_port: "{host}:{port}"
    database: "{database}"
    username: "{username}"
    password: "{password}"
sink:
  type: datahub-rest
  config:
    server: "{datahub_url}"
""",
    "s3": """
source:
  type: s3
  config:
    path_specs:
      - include: "s3://{bucket}/{prefix}/*"
    aws_config:
      aws_access_key_id: "{access_key}"
      aws_secret_access_key: "{secret_key}"
      aws_region: "{region}"
sink:
  type: datahub-rest
  config:
    server: "{datahub_url}"
""",
    "snowflake": """
source:
  type: snowflake
  config:
    account_id: "{account}"
    warehouse: "{warehouse}"
    username: "{username}"
    password: "{password}"
    role: "{role}"
sink:
  type: datahub-rest
  config:
    server: "{datahub_url}"
""",
    "bigquery": """
source:
  type: bigquery
  config:
    project_id: "{project_id}"
    credential:
      project_id: "{project_id}"
      private_key_id: "{private_key_id}"
      private_key: "{private_key}"
      client_email: "{client_email}"
sink:
  type: datahub-rest
  config:
    server: "{datahub_url}"
"""
}

jobs: Dict[str, IngestionStatus] = {}

@app.get("/health")
async def health():
    return {"status": "ok", "service": "ingestion-sidecar"}

@app.post("/ingest", response_model=IngestionStatus)
async def start_ingestion(request: IngestionRequest, background_tasks: BackgroundTasks):
    """Start a DataHub ingestion job"""
    # Use :: as separator since UUIDs contain dashes
    job_id = f"{request.tenant_id}::{request.datasource_id}"
    
    if request.type not in RECIPE_TEMPLATES:
        raise HTTPException(status_code=400, detail=f"Unsupported source type: {request.type}")
    
    jobs[job_id] = IngestionStatus(
        job_id=job_id,
        status="running",
        message="Ingestion started"
    )
    
    background_tasks.add_task(run_ingestion, job_id, request)
    
    return jobs[job_id]

@app.get("/status/{job_id}", response_model=IngestionStatus)
async def get_status(job_id: str):
    """Get ingestion job status"""
    if job_id not in jobs:
        raise HTTPException(status_code=404, detail="Job not found")
    return jobs[job_id]

def run_ingestion(job_id: str, request: IngestionRequest):
    """Run DataHub ingestion in background (synchronous)"""
    import httpx
    
    try:
        # Generate recipe from template
        template = RECIPE_TEMPLATES[request.type]
        config = request.config.copy()
        config["datahub_url"] = os.getenv("DATAHUB_URL", "http://datahub-gms:8080")
        
        recipe_content = template.format(**config)
        
        print(f"Starting ingestion for job {job_id}", flush=True)
        print(f"Recipe:\n{recipe_content}", flush=True)
        
        # Write recipe to temp file
        with tempfile.NamedTemporaryFile(mode='w', suffix='.yaml', delete=False) as f:
            f.write(recipe_content)
            recipe_path = f.name
        
        # Run DataHub ingestion CLI
        result = subprocess.run(
            ["datahub", "ingest", "-c", recipe_path],
            capture_output=True,
            text=True,
            timeout=3600  # 1 hour timeout
        )
        
        print(f"Ingestion stdout: {result.stdout}", flush=True)
        print(f"Ingestion stderr: {result.stderr}", flush=True)
        print(f"Ingestion return code: {result.returncode}", flush=True)
        
        os.unlink(recipe_path)
        
        if result.returncode == 0:
            jobs[job_id].status = "completed"
            jobs[job_id].message = "Ingestion completed successfully"
            # Parse output to get dataset count
            jobs[job_id].datasets_discovered = parse_dataset_count(result.stdout)
        else:
            jobs[job_id].status = "failed"
            # Sanitize stderr to remove null bytes
            stderr_msg = (result.stderr or "Ingestion failed").replace('\x00', '').replace('\0', '')[:500]
            jobs[job_id].message = stderr_msg
            
    except subprocess.TimeoutExpired:
        jobs[job_id].status = "failed"
        jobs[job_id].message = "Ingestion timed out"
    except Exception as e:
        print(f"Ingestion error: {e}", flush=True)
        jobs[job_id].status = "failed"
        jobs[job_id].message = str(e)
    
    # Callback to main service (synchronous to ensure it completes)
    if request.callback_url:
        try:
            print(f"Calling callback URL: {request.callback_url}", flush=True)
            # Sanitize message to remove null bytes and limit length
            message = jobs[job_id].message or ""
            message = message.replace('\x00', '').replace('\0', '')[:500]
            
            callback_data = {
                "job_id": job_id,  # Format: tenant_id::datasource_id
                "status": jobs[job_id].status,
                "message": message,
                "datasets_discovered": jobs[job_id].datasets_discovered
            }
            print(f"Callback data: {callback_data}", flush=True)
            with httpx.Client(timeout=30.0) as client:
                resp = client.post(request.callback_url, json=callback_data)
                print(f"Callback response: {resp.status_code}", flush=True)
        except Exception as e:
            print(f"Callback error: {e}", flush=True)

def parse_dataset_count(output: str) -> int:
    """Parse DataHub CLI output to get dataset count"""
    # Look for patterns like "Emitted 42 datasets"
    import re
    match = re.search(r'Emitted (\d+) datasets?', output)
    if match:
        return int(match.group(1))
    return 0

@app.post("/recipe/validate")
async def validate_recipe(recipe: Dict[str, Any]):
    """Validate a DataHub recipe"""
    try:
        yaml_str = yaml.dump(recipe)
        # Basic validation
        if "source" not in recipe:
            return {"valid": False, "error": "Missing 'source' section"}
        if "sink" not in recipe:
            return {"valid": False, "error": "Missing 'sink' section"}
        return {"valid": True}
    except Exception as e:
        return {"valid": False, "error": str(e)}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8090)
