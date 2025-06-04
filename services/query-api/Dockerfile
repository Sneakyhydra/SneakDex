# Stage 1: Builder stage
FROM python:3.12-slim AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    gcc \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

# Stage 2: Runtime stage
FROM python:3.12-slim

# Set working directory
WORKDIR /app

# Install runtime dependencies and curl for healthcheck
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    curl \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user to run the application
RUN groupadd -r apiuser && useradd -r -g apiuser apiuser

# Create necessary directories with correct permissions
RUN mkdir -p /app/data /home/apiuser/nltk_data && \
    chown -R apiuser:apiuser /app /home/apiuser

# Copy dependencies from builder stage
COPY --from=builder /root/.local /home/apiuser/.local
ENV PATH=/home/apiuser/.local/bin:$PATH
ENV PYTHONPATH=/home/apiuser/.local/lib/python3.12/site-packages
ENV NLTK_DATA=/home/apiuser/nltk_data

# Download NLTK data as root user
RUN PYTHONPATH=/home/apiuser/.local/lib/python3.12/site-packages python -c "import nltk; nltk.download('punkt', download_dir='/home/apiuser/nltk_data'); nltk.download('punkt_tab', download_dir='/home/apiuser/nltk_data'); nltk.download('stopwords', download_dir='/home/apiuser/nltk_data')" && \
    chown -R apiuser:apiuser /home/apiuser/nltk_data

# Copy application code
COPY --chown=apiuser:apiuser . .

# Set environment variables
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV PORT=8000
ENV HOST=0.0.0.0
ENV WORKERS=4
ENV LOG_LEVEL=info

# Create volume mount point with correct permissions
VOLUME ["/app/data"]

# Switch to non-root user
USER apiuser

# Expose the application port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

# Command to run the application with Uvicorn server
CMD ["sh", "-c", "uvicorn api.main:app --host $HOST --port $PORT --workers $WORKERS --log-level $LOG_LEVEL"]

