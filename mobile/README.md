# AlemLive Mobile

Flutter mobile client for AlemLive: auth, rooms, LiveKit calls, chat,
participants, screen sharing, recording controls, reports, transcript tabs and
AI questions.

## Run

Start the local backend stack from the repository root:

```powershell
docker compose up -d --build
```

Then run Flutter:

```powershell
cd mobile
flutter pub get
flutter run
```

For Android emulator, point the app to the host machine:

```powershell
flutter run `
  --dart-define=BACKEND_BASE_URL=http://10.0.2.2:8088 `
  --dart-define=LIVEKIT_URL=ws://10.0.2.2:7880
```

For a real device on the same Wi-Fi network, replace `YOUR_PC_IP`:

```powershell
flutter run `
  --dart-define=BACKEND_BASE_URL=http://YOUR_PC_IP:8088 `
  --dart-define=LIVEKIT_URL=ws://YOUR_PC_IP:7880
```

For Keycloak login on mobile, the app uses the redirect URI
`alemlive://auth/callback`. Keep this URI in the `alemlive` Keycloak client
redirects. If you run on an emulator or a real device, set
`KEYCLOAK_PUBLIC_URL` to an address that the device can open, for example
`http://10.0.2.2:8080` for the Android emulator or `http://YOUR_PC_IP:8080`
for a physical device.

## Packages

- `flutter_riverpod` for state management
- `go_router` for navigation
- `dio` for backend API calls
- `flutter_secure_storage` for auth token storage
- `app_links` and `url_launcher` for Keycloak browser login callbacks
- `livekit_client` for video rooms, chat data messages and media controls
- `permission_handler` for camera, microphone and screen sharing permissions
- `video_player` for report recordings

## Structure

```text
lib/
  app/
  core/
    network/
    storage/
    widgets/
  features/
    ai/
    auth/
    home/
    reports/
    rooms/
```

The app uses clean feature boundaries with `data`, `domain`, and
`presentation` layers where the feature needs them.

## Checks

```powershell
flutter analyze
flutter test
cd android
.\gradlew.bat assembleDebug
```
