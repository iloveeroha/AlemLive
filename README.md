# AlemLive

Видеоконференции на LiveKit с Go backend, React/Vite frontend, AI-отчётами и автоматической записью встреч.

## Структура

```text
AlemLive/
├── backend/                  Go API: LiveKit tokens, отчёты, AI/STT
├── frontend/                 React + Vite UI
├── livekit/                  конфиги LiveKit server и LiveKit Egress
├── keycloak/                 realm import для локальной авторизации
├── docker-compose.yml        весь локальный стек
└── docker-compose.livekit.yml только LiveKit + Redis + MinIO + Egress
```

## Запуск всего стека

```powershell
Copy-Item backend/.env.example backend/.env
docker compose up -d --build
```

Основной compose поднимает frontend, backend, Keycloak, LiveKit, Redis, MinIO и LiveKit Egress.

Локальные адреса:

- Frontend: `http://localhost:5173` или `https://localhost:5174`
- Backend: `http://localhost:8088`
- Keycloak: `http://localhost:8080`
- LiveKit: `ws://localhost:7880`
- MinIO console: `http://localhost:9001` (`alemlive` / `alemlive-secret`)
- Записи: `http://localhost:9000/alemlive-recordings/...`

После завершения встречи LiveKit Egress пишет MP4 в bucket `alemlive-recordings`, backend получает webhook, запускает STT/AI обработку и обновляет отчёт.

## Тестирование по локальной сети

Если к встрече подключается друг с другого компьютера, запусти stack с адресом твоего ПК:

```powershell
$env:LIVEKIT_NODE_IP="YOUR_LAN_IP"
$env:LIVEKIT_PUBLIC_URL="ws://YOUR_LAN_IP:7880"
$env:LIVEKIT_EGRESS_PUBLIC_BASE_URL="http://YOUR_LAN_IP:9000/alemlive-recordings"
docker compose up -d --build
```

Также frontend нужно открывать по HTTPS или localhost, иначе браузер может блокировать камеру и микрофон.

## Отдельный LiveKit stack

```powershell
docker compose -f docker-compose.livekit.yml up -d
```

Он поднимает только LiveKit, Redis, MinIO и Egress. Для backend, запущенного на хосте через `go run`, укажи webhook вроде `http://host.docker.internal:8080/api/livekit/webhook`.

## Важно про LiveKit Cloud

Локальный MinIO не доступен из LiveKit Cloud. Если используешь Cloud-проект вместо локального LiveKit, нужен публичный S3-compatible storage или публично доступный MinIO endpoint.
