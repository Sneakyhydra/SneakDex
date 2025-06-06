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

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    libgomp1 && \  
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create a non-root user to run the application
RUN groupadd -r indexer && useradd -r -g indexer indexer

# Create necessary directories with correct permissions
RUN mkdir -p /app/data /home/indexer/nltk_data && \
    chown -R indexer:indexer /app /home/indexer

# Copy dependencies from builder stage
COPY --from=builder /root/.local /home/indexer/.local
ENV PATH=/home/indexer/.local/bin:$PATH
ENV PYTHONPATH=/home/indexer/.local/lib/python3.12/site-packages
ENV NLTK_DATA=/home/indexer/nltk_data

# Download NLTK data as root user
RUN PYTHONPATH=/home/indexer/.local/lib/python3.12/site-packages python -c "import nltk; nltk.download('punkt', download_dir='/home/indexer/nltk_data'); nltk.download('punkt_tab', download_dir='/home/indexer/nltk_data'); nltk.download('stopwords', download_dir='/home/indexer/nltk_data')" && \
    chown -R indexer:indexer /home/indexer/nltk_data

# Copy application code
COPY --chown=indexer:indexer . .

# Set environment variables
ENV PYTHONPATH=/app
ENV PYTHONUNBUFFERED=1
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHON_ENV=production

# Switch to non-root user
USER indexer

# Create volume mount point with correct permissions
VOLUME ["/app/data"]

# Command to run the application
ENTRYPOINT ["python", "-m", "indexer.main"]

