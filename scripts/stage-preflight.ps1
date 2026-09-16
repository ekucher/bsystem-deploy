#Requires -Version 5.1
<#
.SYNOPSIS
    Stage preflight for the BSYSTEM platform: check that this environment
    could plausibly come up, before anything is started.

.DESCRIPTION
    Reads configuration and reports on it. It never prints a secret's value —
    secrets are reported as present, absent or still-a-placeholder and nothing
    else. That rule is what makes the output safe to paste into a ticket,
    which is what people do with preflight output.

    This is the Windows counterpart of scripts/stage-preflight.sh and applies
    the same checks in the same order, so the two can be compared line by line.

.PARAMETER EnvFile
    Path to the .env file. Defaults to .env in the repository root.

.PARAMETER SkipNetwork
    Skip DNS resolution and TCP connectivity checks.

.PARAMETER SkipDocker
    Skip container runtime checks. For the script's own tests on a machine
    with no Docker daemon — not for skipping an inconvenient failure.

.OUTPUTS
    Exit code 0 when no blocking failure was found, 1 otherwise.
#>
[CmdletBinding()]
param(
    [string]$EnvFile = '.env',
    [switch]$SkipNetwork,
    [switch]$SkipDocker
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$script:Failures = 0
$script:Warnings = 0
$script:Config = @{}

function Write-Pass { param([string]$Message) Write-Host "  PASS  $Message" -ForegroundColor Green }
function Write-Warn { param([string]$Message) Write-Host "  WARN  $Message" -ForegroundColor Yellow; $script:Warnings++ }
function Write-Fail { param([string]$Message) Write-Host "  FAIL  $Message" -ForegroundColor Red; $script:Failures++ }
function Write-Info { param([string]$Message) Write-Host "  ....  $Message" -ForegroundColor DarkGray }
function Write-Section { param([string]$Title) Write-Host ''; Write-Host "== $Title" }

# Read KEY=VALUE pairs without executing the file. Dot-sourcing a .env would
# run whatever is in it, which is an odd thing for a validator to do to a file
# it is about to call untrusted.
function Import-EnvFile {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) { return }
    foreach ($line in Get-Content -LiteralPath $Path) {
        $trimmed = $line.Trim()
        if ($trimmed -eq '' -or $trimmed.StartsWith('#')) { continue }
        $separator = $trimmed.IndexOf('=')
        if ($separator -lt 1) { continue }
        $key = $trimmed.Substring(0, $separator).Trim() -replace '^export\s+', ''
        if ($key -notmatch '^[A-Za-z_][A-Za-z0-9_]*$') { continue }
        $value = $trimmed.Substring($separator + 1)
        if ($value.Length -ge 2) {
            if (($value.StartsWith('"') -and $value.EndsWith('"')) -or
                ($value.StartsWith("'") -and $value.EndsWith("'"))) {
                $value = $value.Substring(1, $value.Length - 2)
            }
        }
        if (-not $script:Config.ContainsKey($key)) { $script:Config[$key] = $value }
    }
}

function Get-ConfigValue {
    param([string]$Name)
    $fromProcess = [Environment]::GetEnvironmentVariable($Name)
    if (-not [string]::IsNullOrWhiteSpace($fromProcess)) { return $fromProcess }
    if ($script:Config.ContainsKey($Name)) { return $script:Config[$Name] }
    return ''
}

# A placeholder that reaches a running service produces a failure far from its
# cause: "password authentication failed" says nothing about the CHANGE_ME two
# files away.
function Test-Placeholder {
    param([string]$Value)
    if ([string]::IsNullOrEmpty($Value)) { return $false }
    $patterns = @('CHANGE_ME', 'change_me', 'REPLACE_ME', 'TODO', 'xxxxx', 'XXXXX',
                  'stage\.example', '\.example($|/|:)', 'your-', '<[^>]+>')
    foreach ($pattern in $patterns) {
        if ($Value -match $pattern) { return $true }
    }
    return $false
}

function Test-RequiredVariable {
    param([string]$Name, [ValidateSet('config', 'secret')][string]$Kind = 'config')
    $value = Get-ConfigValue -Name $Name
    if ([string]::IsNullOrEmpty($value)) { Write-Fail "$Name is required and is not set"; return }
    if (Test-Placeholder -Value $value) { Write-Fail "$Name still holds a placeholder value"; return }
    if ($Kind -eq 'secret') { Write-Pass "$Name is set ($($value.Length) characters)" }
    else { Write-Pass "$Name is set: $value" }
}

function Test-OptionalVariable {
    param([string]$Name, [ValidateSet('config', 'secret')][string]$Kind = 'config')
    $value = Get-ConfigValue -Name $Name
    if ([string]::IsNullOrEmpty($value)) { Write-Info "$Name is not set; the feature it enables stays off"; return }
    if (Test-Placeholder -Value $value) { Write-Fail "$Name is set but still holds a placeholder value"; return }
    if ($Kind -eq 'secret') { Write-Pass "$Name is set ($($value.Length) characters)" }
    else { Write-Pass "$Name is set: $value" }
}

function Test-ConfiguredUrl {
    param([string]$Name, [switch]$RequireTls)
    $value = Get-ConfigValue -Name $Name
    if ([string]::IsNullOrEmpty($value)) { return }

    $uri = $null
    if (-not [Uri]::TryCreate($value, [UriKind]::Absolute, [ref]$uri)) {
        Write-Fail "$Name is not a URL"
        return
    }
    $hostName = $uri.Host
    if ([string]::IsNullOrEmpty($hostName)) { Write-Fail "$Name is not a URL: it has no host"; return }

    if ($RequireTls -and $uri.Scheme -ne 'https') {
        if ($hostName -in @('localhost', '127.0.0.1', '::1')) {
            Write-Warn "$Name is not HTTPS; acceptable only for a local run"
        } else {
            Write-Fail "$Name must be HTTPS in stage (found $($uri.Scheme))"
        }
    }
    Write-Pass "$Name parses as a URL (host $hostName)"

    if ($SkipNetwork) { return }

    $composeServices = @('postgres', 'nats', 'authentik-server', 'integration-core', 'hub')
    try {
        $null = [System.Net.Dns]::GetHostEntry($hostName)
        Write-Pass "$Name resolves: $hostName"
    } catch {
        if ($composeServices -contains $hostName) {
            Write-Info "$Name names a Compose service ($hostName); it resolves inside the stack, not here"
        } else {
            Write-Fail "$Name does not resolve: $hostName"
        }
        return
    }

    $port = $uri.Port
    if ($port -le 0) { return }
    # A read-only TCP connect: nothing is sent, so this cannot alter anything
    # on the far side.
    $client = [System.Net.Sockets.TcpClient]::new()
    try {
        $connect = $client.BeginConnect($hostName, $port, $null, $null)
        if ($connect.AsyncWaitHandle.WaitOne(3000) -and $client.Connected) {
            Write-Pass "$Name accepts TCP on ${hostName}:${port}"
        } else {
            Write-Warn "$Name does not accept TCP on ${hostName}:${port} (a firewall, or not started yet)"
        }
    } catch {
        Write-Warn "$Name does not accept TCP on ${hostName}:${port} (a firewall, or not started yet)"
    } finally {
        $client.Close()
    }
}

Write-Host 'BSYSTEM stage preflight'
Write-Host "reading $EnvFile; secret values are never printed" -ForegroundColor DarkGray

Write-Section 'Configuration file'
if (Test-Path -LiteralPath $EnvFile) {
    Write-Pass "$EnvFile exists"
    Import-EnvFile -Path $EnvFile
} else {
    Write-Warn "$EnvFile not found; relying on the ambient environment"
}

Write-Section 'Required local files'
foreach ($file in @('docker-compose.yml', 'docker-compose.stage.yml', '.env.example')) {
    if (Test-Path -LiteralPath $file) { Write-Pass "$file exists" } else { Write-Fail "$file is missing" }
}

Write-Section 'Platform variables'
Test-RequiredVariable -Name 'POSTGRES_PASSWORD' -Kind secret
Test-RequiredVariable -Name 'AUTHENTIK_SECRET_KEY' -Kind secret
Test-RequiredVariable -Name 'VITE_OIDC_AUTHORITY'
Test-RequiredVariable -Name 'VITE_OIDC_CLIENT_ID'
Test-OptionalVariable -Name 'BIND_ADDRESS'

$secretKey = Get-ConfigValue -Name 'AUTHENTIK_SECRET_KEY'
if ($secretKey -and $secretKey.Length -lt 50) {
    Write-Fail 'AUTHENTIK_SECRET_KEY is shorter than the 50 characters authentik expects'
}

Write-Section 'Identity'
Test-ConfiguredUrl -Name 'VITE_OIDC_AUTHORITY' -RequireTls
$authority = Get-ConfigValue -Name 'VITE_OIDC_AUTHORITY'
if ($authority -and -not $authority.EndsWith('/')) {
    Write-Fail 'VITE_OIDC_AUTHORITY must end with a slash; an issuer mismatch rejects every token with no useful message'
}
Test-OptionalVariable -Name 'AUTHENTIK_USERINFO_URL'
Test-ConfiguredUrl -Name 'AUTHENTIK_USERINFO_URL'

Write-Section 'Source systems'
foreach ($pair in @(
    @{ Url = 'ESPOCRM_URL'; Key = 'ESPOCRM_API_KEY' },
    @{ Url = 'REDMINE_URL'; Key = 'REDMINE_API_KEY' },
    @{ Url = 'OUTLINE_URL'; Key = 'OUTLINE_API_KEY' }
)) {
    Test-OptionalVariable -Name $pair.Url
    Test-ConfiguredUrl -Name $pair.Url -RequireTls
    if (-not [string]::IsNullOrEmpty((Get-ConfigValue -Name $pair.Url))) {
        Test-OptionalVariable -Name $pair.Key -Kind secret
        if ($pair.Url -eq 'OUTLINE_URL' -and [string]::IsNullOrEmpty((Get-ConfigValue -Name 'OUTLINE_API_KEY'))) {
            Write-Fail 'OUTLINE_API_KEY is required when OUTLINE_URL is set: Outline has no anonymous read surface'
        }
    }
}

Write-Section 'Optional subsystems'
Test-OptionalVariable -Name 'OPENSEARCH_URL'
Test-ConfiguredUrl -Name 'OPENSEARCH_URL'
Test-OptionalVariable -Name 'AI_PROVIDER'
if ((Get-ConfigValue -Name 'AI_PROVIDER') -eq 'openai') {
    Test-RequiredVariable -Name 'OPENAI_API_KEY' -Kind secret
}

Write-Section 'Container runtime'
if ($SkipDocker) {
    Write-Info 'SkipDocker was passed; the container runtime was not checked'
} elseif (Get-Command docker -ErrorAction SilentlyContinue) {
    Write-Pass 'docker is on PATH'
    & docker info *> $null
    if ($LASTEXITCODE -eq 0) { Write-Pass 'the Docker daemon is reachable' }
    else { Write-Fail 'docker is installed but the daemon is not reachable' }

    & docker compose version *> $null
    if ($LASTEXITCODE -eq 0) {
        Write-Pass 'docker compose is available'
        & docker compose -f docker-compose.yml -f docker-compose.stage.yml config --quiet *> $null
        if ($LASTEXITCODE -eq 0) { Write-Pass 'the stage Compose configuration renders' }
        else { Write-Fail 'the stage Compose configuration does not render; run the same command to see why' }
    } else {
        Write-Fail 'docker compose is not available'
    }
} else {
    Write-Fail 'docker is not on PATH'
}

Write-Section 'Result'
Write-Host "$($script:Failures) failure(s), $($script:Warnings) warning(s)"
if ($script:Failures -gt 0) {
    Write-Host 'preflight blocked: fix the failures above before starting the stack' -ForegroundColor Red
    exit 1
}
Write-Host 'preflight passed' -ForegroundColor Green
exit 0
