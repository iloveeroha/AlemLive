# AlemLive

Видеоконференции на LiveKit с Go-бэкендом и React/Vite фронтендом.

## Структура

```
AlemLive-Project/
├── backend/    Go API: LiveKit токены, /api/meetings/analysis, /api/ask-ai
├── frontend/   React + Vite (LiveKit room UI)
├── livekit/    конфиг локального LiveKit-сервера для разработки
├── docker-compose.yml          frontend + backend
└── docker-compose.livekit.yml  локальный LiveKit-сервер
```

## Docker (весь стек)

```powershell
Copy-Item backend/.env.example backend/.env
# Заполните backend/.env своими LiveKit Cloud значениями
docker compose up -d --build
```

Приложение будет доступно на `http://localhost:5173`, nginx проксирует `/api` на backend.

Остановить:

```powershell
docker compose down
```

## Локальная разработка

Backend:

```powershell
cd backend
Copy-Item .env.example .env
go run ./cmd/server
```

Frontend (в другом терминале):

```powershell
cd frontend
npm install
npm run dev
```

Vite проксирует `/api` на `http://localhost:8080`.

## Локальный LiveKit (опционально)

```powershell
docker compose -f docker-compose.livekit.yml up -d
```

Локальный LiveKit слушает `ws://localhost:7880`, креды для разработки: `devkey` / `devsecret-local-change-me-32-chars`.
