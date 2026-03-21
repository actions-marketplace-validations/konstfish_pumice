---
title: Mermaid Test
date: 16-08-2025
---

# Mermaid Chart Example

```mermaid
graph TD
    A --> B{B}
    B --> C[C]
    B -->|Test| D[[D]]
    D --> A
```

```mermaid
sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>+B: Hello
    B-->>-A: Hey
```