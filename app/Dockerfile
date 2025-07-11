# Use Node.js 20 slim for smaller image size
FROM node:20-slim

# Set working directory
WORKDIR /app

# Install system dependencies needed for native modules and ONNX runtime
RUN apt-get update && apt-get install -y \
    python3 \
    make \
    g++ \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy package files
COPY package*.json ./
# Copy lock files if they exist
COPY yarn.loc[k] ./
COPY pnpm-lock.yam[l] ./

# Install dependencies based on available lock files
RUN if [ -f yarn.lock ]; then \
        yarn install --frozen-lockfile; \
    elif [ -f pnpm-lock.yaml ]; then \
        corepack enable && pnpm install --frozen-lockfile; \
    elif [ -f package-lock.json ]; then \
        npm ci; \
    else \
        npm install; \
    fi

# Copy source code
COPY . .

# Set environment variables
ENV NODE_ENV=development
ENV NEXT_TELEMETRY_DISABLED=1

# Create next.js cache directory and set permissions
RUN mkdir -p .next && chown -R node:node /app

# Switch to non-root user
USER node

# Expose port
EXPOSE 3000

# Start development server with appropriate package manager
CMD ["sh", "-c", "if [ -f yarn.lock ]; then yarn dev; elif [ -f pnpm-lock.yaml ]; then pnpm dev; else npm run dev; fi"]