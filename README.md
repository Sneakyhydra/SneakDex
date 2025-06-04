# ⚡ SneakDex – Search Engine (MVP to Advanced)

## 🚀 Project Goal

To build a **fully functional, fast, and scalable search engine** from scratch as a personal learning project – incorporating core components like crawling, parsing, indexing, querying, and ranking – while using **modern tools, diverse languages, and distributed systems architecture**.

This project is inspired by early search engines like Google and built with an intent to understand each module of the pipeline while keeping the implementation modular, containerized, and production-oriented.

---

## 🧠 High-Level Objectives

- ✅ **Educational**: Learn core components of search engine design.
- ⚙️ **End-to-End**: From web crawling to UI-based search querying.
- 🧪 **Custom-Built**: Minimal use of premade tools; everything built using libraries.
- 💡 **Performance-Oriented**: Fast and concurrent systems using Go, Rust, and Python.
- 🌍 **Scalable Design**: Kafka, Docker, Redis used to mimic production infra.

---

## 📦 Tech Stack

| Layer            | Tooling & Language                        |
|------------------|-------------------------------------------|
| Frontend         | Next.js (React + Tailwind)                |
| API              | FastAPI (Python)                          |
| Crawler          | Go                                        |
| HTML Parser      | Rust (fast and safe DOM parsing)          |
| Indexer          | Python (TF-IDF, Inverted Index)           |
| Cache            | Redis                                     |
| Messaging        | Apache Kafka                              |
| Containerization | Docker + Docker Compose                   |
| Infra Store (optional) | MongoDB/PostgreSQL (if needed)      |

---

## 📚 Modules Overview

### 1. 🌐 Crawler (Go)
- Fetches pages from the web.
- Emits raw HTML via Kafka.

### 2. 🧼 Parser (Rust)
- Extracts title, body, links from HTML.
- Sends parsed page JSON via Kafka.

### 3. 🧠 Indexer (Python)
- Builds inverted index.
- Calculates TF-IDF.
- Stores index to disk/DB.

### 4. 🧾 Query Engine (FastAPI)
- Loads index into memory.
- Exposes `/search?q=term`.

### 5. ⚡ Cache Layer (Redis)
- Used to cache recent search results.

### 6. 🎯 Frontend (Next.js)
- Modern search UI with real-time results.

---

## 📍 Roadmap Summary

- [x] Define architecture
- [ ] Setup infra (Docker, Kafka, Redis)
- [ ] Implement crawler
- [ ] Parse & clean web pages
- [ ] Index and rank content
- [ ] Expose query API
- [ ] Build frontend
- [ ] Add ranking improvements (PageRank, etc.)