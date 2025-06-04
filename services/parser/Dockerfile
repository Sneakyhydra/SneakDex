# Stage 1: Build the Rust application
FROM rust:1.82-slim as builder

# Set working directory
WORKDIR /usr/src/parser

# Install dependencies
RUN apt-get update && \
    apt-get install -y pkg-config libssl-dev cmake g++ && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create blank project for caching dependencies
RUN USER=root cargo new --bin parser
WORKDIR /usr/src/parser/parser

# Copy manifests for dependency caching
COPY Cargo.toml Cargo.lock* ./

# Build dependencies (this layer is cached as long as your dependencies don't change)
RUN cargo build --release
RUN rm src/*.rs

# Copy actual source code
COPY src ./src

# Build the application with optimizations
RUN touch src/main.rs && cargo build --release

# Stage 2: Create the runtime image
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y ca-certificates libssl3 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create a non-root user and group
RUN groupadd -r parser && useradd -r -g parser parser

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /usr/src/parser/parser/target/release/parser /app/parser

# Set ownership of application files
RUN chown -R parser:parser /app

# Switch to non-root user
USER parser

# Set environment variables
ENV RUST_LOG=info
ENV RUST_BACKTRACE=0

# Command to run
ENTRYPOINT ["/app/parser"]

