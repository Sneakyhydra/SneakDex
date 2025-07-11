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

# Copy manifests for dependency caching
COPY Cargo.toml Cargo.lock* ./

# Build dependencies (this layer is cached as long as your dependencies don't change)
RUN cargo build
RUN rm src/*.rs

# Copy actual source code
COPY src ./src

# Build the application
RUN touch src/main.rs && cargo build

# Copy actual source code
COPY . .

# Set environment variables
ENV RUST_BACKTRACE=1
ENV RUST_LOG=debug

# Expose volume for mounting source code
VOLUME ["/app"]

# Live reload on code changes
CMD ["cargo", "watch", "-x", "run"]
