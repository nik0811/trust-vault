FROM python:3.11-slim

WORKDIR /app

# Install system dependencies for database connectors
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    libpq-dev \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Install DataHub with common source plugins
# Pin to 0.14.0 to match DataHub GMS server version (avoids schema mismatch errors)
# Install pydantic<2 first to avoid compatibility issues
RUN pip install --no-cache-dir 'pydantic>=1.10,<2.0' && \
    pip install --no-cache-dir \
    'acryl-datahub[postgres,mysql,s3,snowflake,bigquery]==0.14.0'

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8090

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8090"]
