# ⚡ SneakDex – Search Engine From Scratch

[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/your-username/sneakdex/actions)
[![Docker](https://img.shields.io/badge/docker-ready-blue)](https://hub.docker.com/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Tech Stack](https://img.shields.io/badge/stack-Go%20%7C%20Rust%20%7C%20Python%20%7C%20Next.js-informational)](#)
[![Status](https://img.shields.io/badge/status-MVP-orange)](#)

---

## 🚀 Project Goal

To build a **fully functional, fast, and scalable search engine** from scratch as a personal learning project — implementing core components like **crawling, parsing, indexing, querying, and ranking** — using **modern tools, diverse languages, and distributed systems architecture**.

Inspired by early search engines like Google, SneakDex is designed to deeply explore each module of the pipeline while keeping the implementation **modular**, **containerized**, and **production-oriented**.

### Why a Search Engine?

- Involves deep understanding of distributed systems, data structures, algorithms, and performance.
- Naturally supports incremental development and testing.
- Applies real-world computer science concepts in an engaging way.
- Fun, challenging, and endlessly extensible (ranking, ML, semantic search... you name it).
- Oh, and I watched "How Google Works" on YouTube... so now here we are.

---

## 🛠️ Initial MVP Architecture

Crawler (Go)
↓ Kafka
Parser (Rust)
↓ Kafka
Indexer (Python, TF-IDF)
↓ Local Index
Query API (FastAPI + Redis Cache)
↓
Frontend (Next.js)

Each component can be developed and tested independently. MVP focuses on verifying the full data pipeline: crawl → parse → index → search.

---

## 🧠 High-Level Objectives

- ✅ **Educational**: Understand core search engine components from scratch.
- ⚙️ **End-to-End**: From raw web pages to a working search UI.
- 🧪 **Custom-Built**: Avoid premade tools — just use libraries.
- ⚡ **Performance-Oriented**: Go, Rust, and Python for concurrency and speed.
- 🌍 **Scalable Design**: Kafka, Docker, and Redis to simulate production infra.

> I wanted to learn Rust and Golang — so I made them the core of this project.  
> Also, I’m new to Docker, so this project comes with caffeine-fueled Googling sessions.  
> Using a search engine to build a search engine? Let's gooo.

---

## 📦 Tech Stack

| Layer                | Tech                               |
| -------------------- | ---------------------------------- |
| Frontend             | Next.js (React + Tailwind CSS)     |
| API                  | FastAPI (Python)                   |
| Crawler              | Go                                 |
| HTML Parser          | Rust (for fast & safe DOM parsing) |
| Indexer              | Python (TF-IDF, inverted index)    |
| Cache                | Redis                              |
| Messaging            | Apache Kafka                       |
| Containerization     | Docker + Docker Compose            |
| Infra Store (future) | MongoDB/PostgreSQL                 |
| Monitoring (future)  | Prometheus, Grafana, ELK Stack     |

---

## 📚 Modules Overview

### 1. 🌐 Crawler (Go)

- Fetches web pages.
- Sends raw HTML to Kafka.

### 2. 🧼 Parser (Rust)

- Parses HTML: title, body, links.
- Sends structured JSON to Kafka.

### 3. 🧠 Indexer (Python)

- Builds inverted index with TF-IDF.
- Stores index locally (disk or DB).

### 4. 🔍 Query API (FastAPI)

- Loads the index into memory.
- Exposes `/search?q=term`.

### 5. ⚡ Cache (Redis)

- Caches frequent queries and responses.

### 6. 🎯 Frontend (Next.js)

- Sleek, reactive search UI.
