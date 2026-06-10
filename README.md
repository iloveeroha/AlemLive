# React + Vite

## Backend

This project now includes a Go backend in `backend/`. It creates short-lived LiveKit tokens so the browser does not need access to `LIVEKIT_API_SECRET`.

## Docker

Run the frontend and backend together:

```powershell
Copy-Item backend/.env.example backend/.env
# Fill backend/.env with your LiveKit Cloud values.
docker compose up -d --build
```

The app will be available at `http://localhost:5173`, and nginx proxies `/api` to the backend container. The backend reads LiveKit settings from `backend/.env`.

Stop the stack:

```powershell
docker compose down
```

## Local Development

Run the backend:

```powershell
cd backend
Copy-Item .env.example .env
go run ./cmd/server
```

In another terminal, run the React app:

```powershell
npm run dev
```

Vite proxies `/api` to `http://localhost:8080`, and the join form can request a token from `POST /api/livekit/token`.

## Optional Local LiveKit

The project can also run a local LiveKit server for development:

```powershell
docker compose -f docker-compose.livekit.yml up -d
```

The local LiveKit server uses `ws://localhost:7880` with dev credentials `devkey` / `devsecret-local-change-me-32-chars`.

This template provides a minimal setup to get React working in Vite with HMR and some ESLint rules.

Currently, two official plugins are available:

- [@vitejs/plugin-react](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react) uses [Oxc](https://oxc.rs)
- [@vitejs/plugin-react-swc](https://github.com/vitejs/vite-plugin-react/blob/main/packages/plugin-react-swc) uses [SWC](https://swc.rs/)

## React Compiler

The React Compiler is not enabled on this template because of its impact on dev & build performances. To add it, see [this documentation](https://react.dev/learn/react-compiler/installation).

## Expanding the ESLint configuration

If you are developing a production application, we recommend using TypeScript with type-aware lint rules enabled. Check out the [TS template](https://github.com/vitejs/vite/tree/main/packages/create-vite/template-react-ts) for information on how to integrate TypeScript and [`typescript-eslint`](https://typescript-eslint.io) in your project.
