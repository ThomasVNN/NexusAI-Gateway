# Local Development Runbook

This guide covers setting up, running, testing, and troubleshooting the NexusAI-Gateway on a local machine.

**Prerequisites**
Before you begin, ensure you have the following installed:
* Docker and Docker Compose
* Go compiler (v1.22.0)
* Node.js (v18 or higher) and NPM

**Local Environment Setup**
1. Clone the repository and navigate to the folder.
2. Initialize local config and dependencies:
   ```bash
   make bootstrap
   ```
   This will automatically copy `.env.example` to `.env` and compile the React application under `web/`.

**Running the Gateway**
Start the database dependencies (PostgreSQL and Redis) and the Go server using the master command:
```bash
make dev
```
The gateway is now listening on `http://localhost:20129`.

**Running the Frontend Separately (Hot Reloading)**
If you are developing the admin React interface and want Vite hot-module reloading:
1. Ensure the Go gateway is running on `http://localhost:20129` (or `make dev-env-up` followed by `make run`).
2. Navigate to the web folder and start the dev server:
   ```bash
   cd web
   npm run dev
   ```
3. Vite will start on `http://localhost:5173` (or similar) and proxy API requests directly to the Go server.

**Testing and Linting**
Before committing any changes, run the validation checks:
* Unit Tests: `make test`
* Lint Checks: `make lint`
