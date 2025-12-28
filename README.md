# 🛡️ Distributed Rate Limiter (Go + Redis)

A high-performance, thread-safe, and distributed rate limiter middleware built with **Golang** and **Redis**. 

It implements the **Token Bucket Algorithm** with **Lua Scripting** to ensure atomicity across distributed instances, preventing race conditions under high concurrency.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![Redis](https://img.shields.io/badge/Redis-v7+-DC382D?style=flat&logo=redis)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

---

## 🚀 Key Features

* **⚡ Distributed & Atomic:** Uses custom **Redis Lua scripts** to perform "check-and-set" operations atomically. This eliminates race conditions where multiple requests could slip through simultaneously (the "Check-then-Act" problem).
* **⏱️ Millisecond Precision:** Unlike standard integer counters, this implementation tracks refill rates at the millisecond level, ensuring smooth traffic flow without "staircase" artifacts.
* **🌊 Token Bucket Algorithm:** Allows for legitimate traffic bursts (configurable capacity) while enforcing a strict average rate over time.
* **📡 Standard Headers:** Returns HTTP headers (`X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After`) to help clients handle backpressure gracefully.
* **🛡️ Fail-Open Design:** Designed to prioritize service availability; if Redis becomes unreachable, traffic is allowed (log-only mode) rather than blocking all users.

---

## 🏗️ Architecture

The system follows the **Middleware Pattern**. Every incoming request passes through the Rate Limiter before reaching the core logic.

```mermaid
sequenceDiagram
    participant User
    participant Go_Server as Go Middleware
    participant Redis

    User->>Go_Server: GET /api/resource
    Go_Server->>Redis: EVAL (Lua Script)
    Note right of Redis: 1. Calculate Refill (ms)<br/>2. Update Tokens<br/>3. Return Status
    Redis-->>Go_Server: {Allowed: true, Tokens: 9}
    
    alt Request Allowed
        Go_Server-->>User: 200 OK (Headers: Remaining=9)
    else Request Blocked
        Go_Server-->>User: 429 Too Many Requests (Retry-After: calculated retry time)
    end