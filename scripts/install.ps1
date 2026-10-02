# AgentSync. Кладёт бинарник agentsync в домашний каталог и добавляет этот каталог в PATH.
# Публикацию не применяет и ИИ-агента не загружает. Пароль в эту инструкцию не входит.
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$dir = Join-Path $env:USERPROFILE ".agentsync"
New-Item -ItemType Directory -Force -Path $dir | Out-Null

$origin = "https://zwarder.ru/api"
if ($env:AGENTSYNC_ORIGIN) {
  $origin = $env:AGENTSYNC_ORIGIN.TrimEnd("/")
}

$arch = $env:PROCESSOR_ARCHITECTURE
if ($env:PROCESSOR_ARCHITEW6432) {
  $arch = $env:PROCESSOR_ARCHITEW6432
}
switch ($arch) {
  "ARM64" { $goarch = "arm64" }
  "AMD64" { $goarch = "amd64" }
  default { throw "Эта архитектура не поддерживается." }
}

$exe = Join-Path $dir "agentsync.exe"
$tmp = Join-Path $dir "agentsync.exe.tmp"
$url = "$origin/cli/agentsync-windows-$goarch.exe"
try {
  Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing
  Move-Item -Force -Path $tmp -Destination $exe
} catch {
  if (Test-Path $tmp) {
    Remove-Item -Force $tmp
  }
  throw "Не удалось скачать agentsync."
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ([string]::IsNullOrEmpty($userPath)) {
  $userPath = ""
}
$items = @($userPath -split ";" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
$found = $false
foreach ($item in $items) {
  if ($item.TrimEnd("\") -ieq $dir) {
    $found = $true
  }
}
if (-not $found) {
  if ($items.Count -eq 0) {
    $updated = $dir
  } else {
    $updated = (($items + $dir) -join ";")
  }
  [Environment]::SetEnvironmentVariable("Path", $updated, "User")
}
$session = @($env:Path -split ";" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
$inSession = $false
foreach ($item in $session) {
  if ($item.TrimEnd("\") -ieq $dir) {
    $inSession = $true
  }
}
if (-not $inSession) {
  $env:Path = "$dir;$env:Path"
}

Write-Output "Команда agentsync добавлена в PATH."
Write-Output "Откройте новое окно терминала, чтобы команда agentsync нашлась."
