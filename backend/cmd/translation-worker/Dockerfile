# Translation Worker - Dockerfile
# Multi-stage build for production deployment

# Stage 1: Build stage
FROM python:3.11-slim AS builder

# Install system dependencies for Japanese NLP and PDF processing
RUN apt-get update && apt-get install -y \
    # MeCab dependencies for fugashi (Japanese tokenizer)
    libmecab-dev \
    mecab-ipadic-utf8 \
    mecab-utils \
    # Build dependencies for Python packages with C extensions
    build-essential \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Copy requirements
COPY requirements.txt .

# Install Python dependencies to a temporary location
# We'll copy only what's needed to the runtime stage
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt


# Stage 2: Runtime stage
FROM python:3.11-slim AS runtime

# Install runtime-only dependencies
RUN apt-get update && apt-get install -y \
    # MeCab runtime libraries (no dev headers needed)
    libmecab2 \
    mecab-ipadic-utf8 \
    mecab-utils \
    # Clean up
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user for security
RUN groupadd -r worker && useradd -r -g worker worker

# Set working directory
WORKDIR /app

# Copy installed Python packages from builder
COPY --from=builder /install /usr/local

# Copy application code
COPY main.py .
COPY config.toml .

# Create volume directories
RUN mkdir -p /watch/incoming /watch/processing /watch/translated \
    /watch/failed /watch/cache /watch/bilingual /watch/checkpoints \
    /config /logs /app/plugins && \
    chown -R worker:worker /watch /config /logs /app

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONPATH=/app

# Switch to non-root user
USER worker

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import sys; sys.exit(0)" || exit 1

# Default command (can be overridden)
CMD ["python", "main.py"]
