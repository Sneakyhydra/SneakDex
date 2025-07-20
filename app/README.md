# 🔍 SneakDex - Scalable Search Engine

A high-performance, full-stack search engine with hybrid search capabilities, text-to-image search, and intelligent caching. Built with Next.js, Qdrant vector database, and PostgreSQL for speed, accuracy, and scalability.

## 📋 Table of Contents

- [🔍 SneakDex - Scalable Search Engine](#-sneakdex---scalable-search-engine)
  - [📋 Table of Contents](#-table-of-contents)
  - [🔍 Overview](#-overview)
    - [Key Capabilities](#key-capabilities)
  - [🏗️ Architecture](#️-architecture)
    - [Components](#components)
    - [Search Flow](#search-flow)
  - [✨ Features](#-features)
    - [Frontend Features](#frontend-features)
    - [Backend Features](#backend-features)
    - [Search Capabilities](#search-capabilities)
    - [Reliability \& Performance](#reliability--performance)
  - [🔧 Prerequisites](#-prerequisites)
  - [⚙️ Configuration](#️-configuration)
    - [Environment Variables](#environment-variables)
    - [Example .env.local](#example-envlocal)
  - [🚀 Usage](#-usage)
    - [Development Setup](#development-setup)
    - [Production Deployment](#production-deployment)
    - [Docker Compose Setup](#docker-compose-setup)
  - [🔗 API Endpoints](#-api-endpoints)
    - [Search API](#search-api)
    - [Image Search API](#image-search-api)
  - [📊 Search Architecture Deep Dive](#-search-architecture-deep-dive)
    - [Hybrid Search Implementation](#hybrid-search-implementation)
    - [Vector Embeddings](#vector-embeddings)
    - [Caching Strategy](#caching-strategy)
    - [Fallback Mechanisms](#fallback-mechanisms)
  - [🛠️ Deployment](#️-deployment)
    - [Scaling Considerations](#scaling-considerations)
  - [🐛 Troubleshooting](#-troubleshooting)
    - [Common Issues](#common-issues)
    - [Performance Tuning](#performance-tuning)
  - [🔒 Security](#-security)
  - [📜 License](#-license)

## 🔍 Overview

SneakDex is a modern, scalable search engine that combines the power of vector embeddings with traditional full-text search. It features a clean Next.js frontend with both web search and text-to-image search capabilities, backed by a sophisticated hybrid search system.

### Key Capabilities

- **Hybrid Search**: Combines Qdrant vector search (75% weight) with PostgreSQL full-text search (25% weight)
- **Text-to-Image Search**: Pure vector search for image discovery using semantic embeddings
- **Intelligent Caching**: Redis-based result caching with Upstash for optimal performance
- **Robust Fallbacks**: Multiple fallback layers ensure search availability
- **Real-time Results**: Fast, responsive search with sub-second response times
- **Scalable Architecture**: Designed to handle millions of documents and concurrent users

## 🏗️ Architecture

```
                                    [Redis Cache] ← [Upstash]
                                          ↕
[Frontend] → [Next.js API Routes] → [Hybrid Search Engine] → [PostgreSQL]
                                          ↓
                                    [ML Embeddings] ← [MiniLM-L12-v2 + HuggingFace API]
                                          ↓
                                      [Qdrant]
```

### Components

- **🎨 Next.js Frontend**: Homepage and search interface with responsive design
- **🔧 Next.js Backend**: API routes for search functionality and data processing
- **🧠 Hybrid Search Engine**: Intelligent combination of vector and keyword search
- **📊 Qdrant Vector DB**: High-performance vector similarity search
- **🗄️ PostgreSQL**: Full-text search and structured data storage
- **⚡ Redis Cache**: Fast result caching with Upstash integration
- **🤖 ML Pipeline**: Text embeddings using MiniLM-L12-v2 with HuggingFace fallback

### Search Flow

1. **Query Processing**: User query is processed and embedded
2. **Hybrid Search**: Parallel execution of vector (75%) and full-text (25%) search
3. **Result Fusion**: Intelligent merging of results based on relevance scores
4. **Caching**: Results cached in Redis for subsequent requests
5. **Fallback Handling**: Automatic fallback to alternative search methods if needed

## ✨ Features

### Frontend Features

- ✅ **Homepage**: Clean, responsive landing page
- ✅ **Web Search**: Traditional text-based search interface
- ✅ **Image Search**: Text-to-image semantic search
- ✅ **Real-time Results**: Fast, responsive search experience
- ✅ **Mobile Optimized**: Works seamlessly across all devices

### Backend Features

- ✅ **RESTful APIs**: Clean API design for search operations
- ✅ **Hybrid Search**: Advanced search combining multiple techniques
- ✅ **Smart Caching**: Intelligent result caching with TTL management
- ✅ **Error Handling**: Comprehensive error handling and logging
- ✅ **Rate Limiting**: Built-in protection against abuse

### Search Capabilities

- ✅ **Vector Search**: Semantic similarity using MiniLM-L12-v2 embeddings
- ✅ **Full-text Search**: Traditional keyword-based search with PostgreSQL
- ✅ **Image Search**: Text-to-image search using vector embeddings
- ✅ **Result Ranking**: Sophisticated scoring and ranking algorithms
- ✅ **Query Expansion**: Automatic query enhancement for better results

### Reliability & Performance

- ✅ **Fallback**: Vector → Payload fallback
- ✅ **Distributed Caching**: Redis-based caching with Upstash
- ✅ **Connection Pooling**: Optimized database connections
- ✅ **Graceful Degradation**: System continues working even if components fail

## 🔧 Prerequisites

- **Next.js**: >= 15.4.1
- **PostgreSQL**: >= 14.0
- **Qdrant**: == 1.15.0
- **Redis**: >= 6.0 (or Upstash account)
- **Docker**: Optional, for containerized deployment

## ⚙️ Configuration

### Environment Variables

| Variable                        | Default                 | Description                                 |
| ------------------------------- | ----------------------- | ------------------------------------------- |
| `QDRANT_URL`                    | `http://localhost:6333` | Qdrant server URL                           |
| `QDRANT_API_KEY`                | -                       | Qdrant API key (optional)                   |
| `SUPABASE_URL`                  | -                       | SUPABASE server URL                         |
| `SUPABASE_API_KEY`              | -                       | SUPABASE API key                            |
| `QDRANT_COLLECTION_NAME`        | -                       | Qdrant documents collection name            |
| `QDRANT_COLLECTION_NAME_IMAGES` | -                       | Qdrant images collection name               |
| `UPSTASH_REDIS_REST_URL`        | -                       | Upstash Redis REST URL                      |
| `UPSTASH_REDIS_REST_TOKEN`      | -                       | Upstash Redis REST token                    |
| `HUGGINGFACE_API_KEY`           | -                       | HuggingFace API key for embeddings fallback |

### Example .env.local

```env
# Qdrant Vector Database
QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=your-qdrant-api-key
QDRANT_COLLECTION_NAME=sneakdex
QDRANT_COLLECTION_NAME_IMAGES=sneakdex-images

# Redis Cache (Upstash)
UPSTASH_REDIS_REST_URL=https://your-redis.upstash.io
UPSTASH_REDIS_REST_TOKEN=your-redis-token

# Supabase Postgres DB
SUPABASE_URL=your-supabase-url
SUPABASE_API_KEY=your-supabase-api-key

# ML & Embeddings
HUGGINGFACE_API_KEY=your-huggingface-token
```

## 🚀 Usage

### Development Setup

```bash
# Clone the repository
git clone https://github.com/Sneakyhydra/SneakDex.git
cd sneakdex

# Install dependencies
npm install

# Set up environment variables
cp .env.example .env.local
# Edit .env.local with your configuration

# Start development server
npm run dev
```

Visit `http://localhost:3000` to access the application.

### Production Deployment

```bash
# Build for production
npm run build

# Start production server
npm start
```

### Docker Compose Setup

```yaml
app:
  build:
    context: ./app
  container_name: sneakdex-app
  environment:
    - NODE_ENV=development
    - CHOKIDAR_USEPOLLING=true
    - WATCHPACK_POLLING=true
  env_file:
    - ./app/.env.local
  ports:
    - "3000:3000"
  volumes:
    - ./app:/app
    - /app/node_modules
    - /app/.next
  networks:
    - monitoring
    - sneakdex-network
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:3000/api/health"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 30s
  restart: unless-stopped
```

or for production -
```yaml
app:
  build:
    context: ./app
    dockerfile: Dockerfile.prod
    target: production
  container_name: sneakdex-app
  environment:
    - NODE_ENV=production
    - NEXT_TELEMETRY_DISABLED=1
    - PORT=3000
    - HOSTNAME=0.0.0.0
  env_file:
    - ./app/.env.local
  ports:
    - "3000:3000"
  networks:
    - monitoring
    - sneakdex-network
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:3000/api/health"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 20s
  restart: unless-stopped
```

## 🔗 API Endpoints

### Search API

**POST** `/api/search`

```bash
curl -X POST http://localhost:3000/api/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "artificial intelligence machine learning",
    "top_k": 50,
    "useEmbeddings": true,
  }'
```

Response:
```json
{
    "source": "Qdrant vector + Supabase",
  "results": [
    {
      "id": "doc-123",
      "hybridScore": 0.95,
      "qdrantScore": 0.95,
      "pgScore": 0.95,
      "payload": {
        "url": "string",
        "title": "string",
        "description": "string",
        "headings": [],
        "images": [],
        "language": "string",
        "timestamp": "string",
        "content_type": "string",
        "text_snippet": "string",
        "[key]": "[any]"
      },
      "url": "https://example.com/cat-meme",
      "title" : "meme"
    }
  ],
  "query": "memes",
  "top_k": 50,
  "useEmbeddings": true,
  "total_available": {
    "qdrant": 10000,
    "postgres": 10000
  }
}
```

### Image Search API

**POST** `/api/search-images`

```bash
curl -X POST http://localhost:3000/api/search-images \
  -H "Content-Type: application/json" \
  -d '{
    "query": "sunset over mountains",
    "top_k": 50,
    "useEmbeddings": true,
  }'
```

## 📊 Search Architecture Deep Dive

### Hybrid Search Implementation

The hybrid search combines two complementary approaches:

1. **Vector Search (75% weight)**: Uses MiniLM-L12-v2 embeddings for semantic similarity
2. **Full-text Search (25% weight)**: PostgreSQL's tsvector search for exact matches

Results are merged using a weighted scoring system that balances semantic understanding with keyword precision.

### Vector Embeddings

- **Primary**: Local MiniLM-L12-v2 model for fast, offline embeddings
- **Fallback**: HuggingFace API for high availability
- **Dimensions**: 384-dimensional vectors stored in Qdrant
- **Similarity Metric**: Cosine similarity for optimal semantic search

### Caching Strategy

- **L1 Cache**: In-memory embeddings caching for immediate repeated queries
- **L2 Cache**: Redis/Upstash for persistent caching across instances
- **Cache Key**: query + parameters for efficient lookups
- **TTL**: Expiration (2 hour)

### Fallback Mechanisms

1. **Vector Search** → 2. **Qdrant Payload Search** + 3. **PostgreSQL Tsvector Search**

## 🛠️ Deployment

### Scaling Considerations

- **Horizontal Scaling**: Deploy multiple Next.js instances behind a load balancer
- **Database Scaling**: Use read replicas for PostgreSQL, Qdrant clustering
- **Cache Scaling**: Redis Cluster or Upstash scaling for high throughput
- **CDN**: Use CDN for static assets and API response caching

## 🐛 Troubleshooting

### Common Issues

| Symptom                | Solution                                             |
| ---------------------- | ---------------------------------------------------- |
| Slow search responses  | Check Qdrant connection, increase cache TTL          |
| Embedding API failures | Verify HuggingFace API key, check fallback           |
| Cache misses           | Verify Redis connection, check Upstash configuration |
| Low search quality     | Adjust hybrid search weights, retrain embeddings     |

### Performance Tuning

## 🔒 Security

- ✅ **Input Validation**: Comprehensive query sanitization
- ✅ **Environment Security**: Sensitive data in environment variables only

## 📜 License

MIT License - feel free to use, modify, and contribute to this project.