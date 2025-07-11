FROM rust:1.82-slim

# Set working directory
WORKDIR /app

# Install development dependencies
RUN apt-get update && \
    apt-get install -y pkg-config libssl-dev cmake g++ curl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Install cargo-watch for live reload
RUN cargo install cargo-watch

# Copy manifests first for caching deps
COPY Cargo.toml Cargo.lock* ./

# Create dummy main.rs to build deps
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build
RUN rm -rf src/*

# Copy actual source code
COPY . .

# Set environment variables
ENV RUST_BACKTRACE=1
ENV RUST_LOG=debug

# Expose volume for mounting source code
VOLUME ["/app"]

# Live reload on code changes
CMD ["cargo", "watch", "-x", "run"]
