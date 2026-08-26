# Runtime process ownership

| Process | Composition root | Owns | Does not own |
| --- | --- | --- | --- |
| Web | `apps/web` | Browser bootstrap and delivery | Product feature duplication |
| Desktop | `apps/desktop/src-tauri` | Native lifecycle/capabilities | Server business rules |
| Core API | `services/core/cmd/api` | REST composition and request lifecycle | Long-running work |
| Core worker | `services/core/cmd/worker` | Temporal registration and business activities | Public workflow definitions |
| Migration utility | `services/core/cmd/migrate` | Ordered PostgreSQL schema changes | Runtime auto-migration |
| Document worker | Python `document-worker` | Parsing/OCR/normalization/chunking activities | Model distribution |
| AI worker | Python `ai-worker` | Provider-backed embedding/extraction activities | Authorization policy |

Processes share contracts, not private implementation types. A new process requires an isolation, scaling, runtime, or fault-containment reason and an ADR.
