# FlowForge: Real-Time Multi-Tenant DAG Execution Platform

FlowForge is a production-grade, multi-tenant workflow orchestration platform. It is built with a highly concurrent Go backend utilizing Clean Architecture principles, combined with a modern, reactive Vue 3 dashboard frontend.

---

## 🌟 Core Features

* **Concurrent DAG Execution Engine**: Runs independent tasks concurrently using goroutines, sync primitives, and custom retries with exponential backoffs, while executing dependent tasks sequentially.
* **Topological Sort Validation**: Utilizes Kahn's BFS algorithm to detect cycles (circular dependencies) and validate the workflow graph before execution.
* **Strict Multi-Tenant Isolation**: Implements a custom GORM plugin ([TenantIsolationPlugin](file:///home/rafia/Documents/rnd/flow-forger/backend/repository/tenant_plugin.go)) that intercepts database queries, updates, deletes, and insertions to automatically inject the `tenant_id` context filter at the persistence layer.
* **Real-Time SSE Event Stream**: Streams execution stats and logs to the frontend in real-time via Server-Sent Events (SSE).
* **AI-Powered Diagnostics**: Integrates OpenRouter structured completion API to automatically diagnose failed workflow runs based on runtime errors, stepping in with helpful remediation suggestions.
* **Dual-Database Support**: Decouples database initialization using GORM dialectors, allowing lightning-fast unit and integration testing via in-memory SQLite while using PostgreSQL for local dev and production runtimes.
* **Environment Configuration**: Manages sensitive API keys and database credentials via secure `.env` files.

---

## ⚡ Quick Setup (Docker Compose)

The easiest way to run the entire stack (Go Backend, PostgreSQL, Vue 3 Frontend) is using Docker Compose:

### 1. Copy the Environment Template
```bash
cp .env.example .env
```

### 2. Configure OpenRouter API Key (Optional)
Edit the [.env](file:///home/rafia/Documents/rnd/flow-forger/.env) file in the root directory and add your OpenRouter API Key:
```env
OPENROUTER_API_KEY=your-api-key-here
```
*(If no API key is provided, the platform automatically falls back to offline mock diagnostics).*

### 3. Start the Containers
```bash
docker compose up --build -d
```

### 4. Access the Platform
* **Frontend Web Dashboard**: Open [http://localhost:82](http://localhost:82) in your browser.
* **Backend REST API**: Accessible at [http://localhost:8080/api](http://localhost:8080/api).
* **Default Seeding Credentials**:
  * **Tenant ID**: `tenant-a`
  * **Email**: `editor@tenant-a.com`
  * **Password**: `password123`
  * *(The backend automatically seeds a default ETL pipeline, users, and tenants on startup!)*

---

## 🏗️ Architecture Design Summary

FlowForge follows a clean, decoupled repository layout separated into Go backend and Vue 3 frontend codebases.

### 1. Go Backend (Clean Architecture)
Located in the [backend/](file:///home/rafia/Documents/rnd/flow-forger/backend) directory:

* **Domain Model Layer ([domain/](file:///home/rafia/Documents/rnd/flow-forger/backend/domain))**: Defines the core data schemas (`Tenant`, `User`, `Workflow`, `WorkflowRun`, `ExecutionLog`) and application-level context keys. Free of external library dependencies.
* **Repository Layer ([repository/](file:///home/rafia/Documents/rnd/flow-forger/backend/repository))**:
  * [db.go](file:///home/rafia/Documents/rnd/flow-forger/backend/repository/db.go): Manages GORM migrations and multi-dialector setup.
  * [tenant_plugin.go](file:///home/rafia/Documents/rnd/flow-forger/backend/repository/tenant_plugin.go): The multi-tenant security plugin enforcing boundaries.
  * [run_repository.go](file:///home/rafia/Documents/rnd/flow-forger/backend/repository/run_repository.go): Handles CRUD operations for runs, execution logs, and transactions.
* **Usecase Layer ([usecase/](file:///home/rafia/Documents/rnd/flow-forger/backend/usecase))**:
  * [dag.go](file:///home/rafia/Documents/rnd/flow-forger/backend/usecase/dag.go): Kahn's BFS topological sorting logic.
  * [engine.go](file:///home/rafia/Documents/rnd/flow-forger/backend/usecase/engine.go): Concurrent workflow step execution logic.
  * [error_analyzer.go](file:///home/rafia/Documents/rnd/flow-forger/backend/usecase/error_analyzer.go): Integrates OpenRouter for structured AI diagnostics.
* **Delivery/HTTP Layer ([delivery/](file:///home/rafia/Documents/rnd/flow-forger/backend/delivery))**: Implements Fiber routing, JSON payload validation, JWT authentication middleware, and Server-Sent Event (SSE) streaming controllers.

### 2. Vue 3 Frontend (Composition API & Pinia)
Located in the [frontend/](file:///home/rafia/Documents/rnd/flow-forger/frontend) directory:

* **API Client Layer ([src/api/](file:///home/rafia/Documents/rnd/flow-forger/frontend/src/api))**: Strongly typed Axios instances with automatic JWT header mapping and request interceptors.
* **State Management ([src/stores/](file:///home/rafia/Documents/rnd/flow-forger/frontend/src/stores))**: Global state containers handling authentication, cache management, and real-time SSE event binding.
* **Components ([src/components/](file:///home/rafia/Documents/rnd/flow-forger/frontend/src/components))**:
  * `DagViewer.vue`: Renders interactive workflow execution graphs using Vue Flow.
  * `HealthPanel.vue`: Visualizes active runs, execution durations, and historical success rates.
  * `StepLogViewer.vue`: Collapsible step execution terminal logs displaying embedded AI diagnostics.

---

## 🛠️ Local Development (Manual)

If you prefer to run the backend and frontend services directly on your host machine:

### Go Backend Setup
```bash
cd backend
cp ../.env .env            # Copy env file to backend directory
go test ./... -v           # Run backend unit/integration tests
go run cmd/server/main.go  # Start server locally on port 8080
```

### Vue 3 Frontend Setup
```bash
cd frontend
npm install
npm run build              # Run production bundle verification
npm run dev                # Start Vite dev server on port 5173
```
