param(
  [string]$Ip = "",
  [switch]$NoDocker,
  [switch]$NoBuild
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptDir
Set-Location $repoRoot

function Test-IPv4 {
  param([string]$Value)

  $parsed = $null
  if (-not [System.Net.IPAddress]::TryParse($Value, [ref]$parsed)) {
    return $false
  }

  return $parsed.AddressFamily -eq [System.Net.Sockets.AddressFamily]::InterNetwork
}

function Get-CurrentIPv4 {
  function Test-IsVirtualAdapter {
    param([string]$Alias)

    return $Alias -match "vEthernet|WSL|Docker|Default Switch|Hyper-V|VirtualBox|VMware|Loopback|Bluetooth|Radmin|ZeroTier|Tailscale|Npcap"
  }

  $items = Get-NetIPAddress -AddressFamily IPv4 |
    Where-Object {
      $_.IPAddress -notlike "127.*" -and
      $_.IPAddress -notlike "169.254.*" -and
      $_.PrefixOrigin -ne "WellKnown" -and
      $_.AddressState -eq "Preferred"
    } |
    Select-Object IPAddress, InterfaceAlias

  $physical = $items | Where-Object {
    $_.InterfaceAlias -match "Wi-Fi|Wireless|Ethernet" -and
    -not (Test-IsVirtualAdapter $_.InterfaceAlias)
  }

  if ($physical) {
    return ($physical | Select-Object -First 1).IPAddress
  }

  $fallback = $items | Where-Object {
    -not (Test-IsVirtualAdapter $_.InterfaceAlias)
  }

  if ($fallback) {
    return ($fallback | Select-Object -First 1).IPAddress
  }

  throw "Could not find a usable IPv4 address. Pass it manually: .\scripts\update-local-ip.ps1 -Ip 10.x.x.x"
}

function Set-EnvValue {
  param(
    [string]$Path,
    [string]$Key,
    [string]$Value
  )

  if (-not (Test-Path -LiteralPath $Path)) {
    throw "Missing env file: $Path"
  }

  $lines = [System.Collections.Generic.List[string]]::new()
  $lines.AddRange([System.IO.File]::ReadAllLines((Resolve-Path -LiteralPath $Path)))

  $pattern = "^\s*$([regex]::Escape($Key))="
  $next = "$Key=$Value"
  $updated = $false

  for ($i = 0; $i -lt $lines.Count; $i++) {
    if ($lines[$i] -match $pattern) {
      $lines[$i] = $next
      $updated = $true
      break
    }
  }

  if (-not $updated) {
    if ($lines.Count -gt 0 -and $lines[$lines.Count - 1].Trim() -ne "") {
      $lines.Add("")
    }
    $lines.Add($next)
  }

  $utf8NoBom = [System.Text.UTF8Encoding]::new($false)
  [System.IO.File]::WriteAllLines((Resolve-Path -LiteralPath $Path), $lines, $utf8NoBom)
}

function Get-EnvValue {
  param(
    [string]$Path,
    [string]$Key,
    [string]$DefaultValue = ""
  )

  if (-not (Test-Path -LiteralPath $Path)) {
    return $DefaultValue
  }

  $pattern = "^\s*$([regex]::Escape($Key))=(.*)$"
  foreach ($line in [System.IO.File]::ReadAllLines((Resolve-Path -LiteralPath $Path))) {
    if ($line -match $pattern) {
      return $Matches[1].Trim()
    }
  }

  return $DefaultValue
}

function Add-UniqueValue {
  param(
    [object[]]$Values,
    [string]$Value
  )

  if ($Values -contains $Value) {
    return $Values
  }

  return @($Values) + $Value
}

function Sync-KeycloakRealmFile {
  param(
    [string]$Path,
    [string]$ClientId,
    [string[]]$RedirectUris
  )

  if (-not (Test-Path -LiteralPath $Path)) {
    Write-Host "Skipped Keycloak realm file update: missing $Path" -ForegroundColor Yellow
    return
  }

  $realm = Get-Content -Raw -LiteralPath $Path | ConvertFrom-Json
  $client = $realm.clients | Where-Object { $_.clientId -eq $ClientId } | Select-Object -First 1
  if (-not $client) {
    Write-Host "Skipped Keycloak realm file update: client '$ClientId' was not found" -ForegroundColor Yellow
    return
  }

  $uris = @($client.redirectUris)
  foreach ($uri in $RedirectUris) {
    $uris = Add-UniqueValue $uris $uri
  }
  $client.redirectUris = @($uris)

  $json = $realm | ConvertTo-Json -Depth 20
  $utf8NoBom = [System.Text.UTF8Encoding]::new($false)
  [System.IO.File]::WriteAllText((Resolve-Path -LiteralPath $Path), $json + [Environment]::NewLine, $utf8NoBom)
}

function Sync-RunningKeycloakClient {
  param(
    [string]$BaseUrl,
    [string]$Realm,
    [string]$ClientId,
    [string[]]$RedirectUris
  )

  $adminUser = Get-EnvValue ".env" "KEYCLOAK_ADMIN_USERNAME" "admin"
  $adminPassword = Get-EnvValue ".env" "KEYCLOAK_ADMIN_PASSWORD" "admin"
  $tokenUrl = "$BaseUrl/realms/master/protocol/openid-connect/token"

  try {
    $tokenResponse = Invoke-RestMethod -Method Post -Uri $tokenUrl -ContentType "application/x-www-form-urlencoded" -Body @{
      client_id = "admin-cli"
      username = $adminUser
      password = $adminPassword
      grant_type = "password"
    }

    $headers = @{ Authorization = "Bearer $($tokenResponse.access_token)" }
    $clientsUrl = "$BaseUrl/admin/realms/$Realm/clients?clientId=$([uri]::EscapeDataString($ClientId))"
    $clients = Invoke-RestMethod -Method Get -Uri $clientsUrl -Headers $headers
    $client = @($clients) | Select-Object -First 1
    if (-not $client) {
      Write-Host "Skipped running Keycloak update: client '$ClientId' was not found" -ForegroundColor Yellow
      return
    }

    $clientUrl = "$BaseUrl/admin/realms/$Realm/clients/$($client.id)"
    $clientDetails = Invoke-RestMethod -Method Get -Uri $clientUrl -Headers $headers
    $uris = @($clientDetails.redirectUris)
    foreach ($uri in $RedirectUris) {
      $uris = Add-UniqueValue $uris $uri
    }
    $clientDetails.redirectUris = @($uris)

    $body = $clientDetails | ConvertTo-Json -Depth 20
    Invoke-RestMethod -Method Put -Uri $clientUrl -Headers $headers -ContentType "application/json" -Body $body | Out-Null
    Write-Host "Updated running Keycloak client redirect URIs" -ForegroundColor Green
  } catch {
    Write-Host "Could not update running Keycloak client automatically: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host "Open Keycloak admin and add redirect URIs manually:" -ForegroundColor Yellow
    foreach ($uri in $RedirectUris) {
      Write-Host "  $uri" -ForegroundColor Yellow
    }
  }
}

if ($Ip.Trim()) {
  $targetIp = $Ip.Trim()
  if (-not (Test-IPv4 $targetIp)) {
    throw "Invalid IPv4 address: $targetIp"
  }
} else {
  $targetIp = Get-CurrentIPv4
}

$frontendHttp = "http://${targetIp}:5173"
$frontendHttps = "https://${targetIp}:5174"
$keycloakBase = "http://${targetIp}:8080"
$minioRecordings = "http://${targetIp}:9000/alemlive-recordings"
$keycloakRealm = Get-EnvValue ".env" "KEYCLOAK_REALM" "alemlive"
$keycloakClientId = Get-EnvValue ".env" "KEYCLOAK_CLIENT_ID" "alemlive"
$redirectUris = @(
  "http://localhost:5173/*",
  "https://localhost:5174/*",
  "http://127.0.0.1:5173/*",
  "https://127.0.0.1:5174/*",
  "$frontendHttp/*",
  "$frontendHttps/*"
)

Write-Host "Using local IP: $targetIp" -ForegroundColor Cyan

Set-EnvValue ".env" "APP_HOST" $targetIp
Set-EnvValue ".env" "LIVEKIT_NODE_IP" $targetIp
Set-EnvValue ".env" "KEYCLOAK_PUBLIC_URL" $keycloakBase
Set-EnvValue ".env" "LIVEKIT_EGRESS_PUBLIC_BASE_URL" $minioRecordings

Set-EnvValue "backend/.env" "ALLOWED_ORIGINS" "http://localhost:5173,https://localhost:5174,$frontendHttp,$frontendHttps"
Set-EnvValue "backend/.env" "KEYCLOAK_ISSUER_URL" "$keycloakBase/realms/$keycloakRealm"
Set-EnvValue "backend/.env" "KEYCLOAK_JWKS_URL" "$keycloakBase/realms/$keycloakRealm/protocol/openid-connect/certs"
Set-EnvValue "backend/.env" "KEYCLOAK_TOKEN_URL" "$keycloakBase/realms/$keycloakRealm/protocol/openid-connect/token"
Set-EnvValue "backend/.env" "LIVEKIT_EGRESS_PUBLIC_BASE_URL" $minioRecordings

Sync-KeycloakRealmFile "keycloak/alemlive-realm.json" $keycloakClientId $redirectUris

Write-Host "Updated .env, backend/.env and Keycloak realm redirects" -ForegroundColor Green

if ($NoDocker) {
  Sync-RunningKeycloakClient $keycloakBase $keycloakRealm $keycloakClientId $redirectUris
  Write-Host "Skipped Docker restart because -NoDocker was passed." -ForegroundColor Yellow
  Write-Host "Open: $frontendHttps"
  exit 0
}

docker compose config --quiet

if (-not $NoBuild) {
  Write-Host "Rebuilding frontend certificate/build for $targetIp..." -ForegroundColor Cyan
  docker compose build frontend
}

Write-Host "Recreating services with the new IP..." -ForegroundColor Cyan
docker compose up -d --force-recreate frontend backend keycloak livekit livekit-egress

Start-Sleep -Seconds 5
Sync-RunningKeycloakClient $keycloakBase $keycloakRealm $keycloakClientId $redirectUris

Write-Host ""
Write-Host "Done." -ForegroundColor Green
Write-Host "Frontend: $frontendHttps"
Write-Host "Backend:  http://${targetIp}:8088"
Write-Host "Keycloak: $keycloakBase"
