FROM rust:1.82-slim

# Set working directory
WORKDIR /app

# Install system dependencies and cargo tools in one layer
RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    cmake \
    g++ \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && cargo install cargo-watch cargo-chef

# Use cargo-chef for better dependency caching
COPY Cargo.toml Cargo.lock ./

# Create dummy source to satisfy cargo-chef
RUN mkdir src && echo "fn main() {}" > src/main.rs

RUN cargo chef prepare --recipe-path recipe.json
RUN cargo chef cook --recipe-path recipe.json

# Clean up dummy source
RUN rm -rf src

# Development environment with optimizations
ENV RUST_BACKTRACE=1 \
    RUST_LOG=info \
    CARGO_INCREMENTAL=1 \
    CARGO_TARGET_DIR=/app/target

# Live reload on code changes
CMD ["cargo", "watch", "-x", "run"]
