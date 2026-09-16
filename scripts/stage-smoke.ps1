#Requires -Version 5.1
<#
.SYNOPSIS
    Non-destructive stage acceptance pass over a running BSYSTEM deployment.

.DESCRIPTION
    Every check is a GET. Nothing is created, updated or deleted, so this can
    be run against an environment someone else is using.

    Tokens are read from the environment and never printed, never written to
    the report and never included in a diagnostic. A check needing a token that
    is not available is SKIPped with the reason, rather than failing the whole
    unauthenticated run — a runner that goes red for a missing optional token
    teaches people to ignore it.

    This is the Windows counterpart of scripts/stage-smoke.sh and performs the
    same checks in the same order.

.OUTPUTS
    artifacts/stage-acceptance.json and artifacts/stage-acceptance.md.
    Exit code 0 when no check failed, 1 otherwise.
#>
[CmdletBinding()]
param(
    [string]$CoreUrl = $(if ($env:CORE_URL) { $env:CORE_URL } else { 'http://127.0.0.1:8080' }),
    [string]$HubUrl = $(if ($env:HUB_URL) { $env:HUB_URL } else { 'http://127.0.0.1:8081' }),
    [string]$AuthentikUrl = $env:AUTHENTIK_URL,
    [string]$ArtifactsDir = $(if ($env:ARTIFACTS_DIR) { $env:ARTIFACTS_DIR } else { 'artifacts' }),
    [int]$TimeoutSeconds = $(if ($env:TIMEOUT) { [int]$env:TIMEOUT } else { 10 })
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$humanToken = $env:BSYSTEM_HUMAN_TOKEN
$serviceToken = $env:BSYSTEM_SERVICE_TOKEN
$results = [System.Collections.Generic.List[object]]::new()
$startedAt = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')

function Add-Result {
    param(
        [string]$Service, [string]$Check,
        [ValidateSet('PASS', 'FAIL', 'SKIP', 'BLOCKED')][string]$Result,
        [int]$DurationMs = 0, [string]$RequestId = '', [string]$Message = ''
    )
    $results.Add([pscustomobject]@{
        service = $Service; check = $Check; result = $Result
        duration_ms = $DurationMs; request_id = $RequestId; message = $Message
    })
    $colour = switch ($Result) { 'PASS' { 'Green' } 'FAIL' { 'Red' } 'SKIP' { 'Yellow' } 'BLOCKED' { 'DarkGray' } }
    $line = '  {0,-7} {1,-22} {2}' -f $Result, $Service, $Check
    if ($Message -and $Result -ne 'PASS') { $line += " — $Message" }
    Write-Host $line -ForegroundColor $colour
}

function Invoke-Check {
    param(
        [string]$Service, [string]$Check, [string]$Url,
        [int[]]$Expected, [ValidateSet('none', 'human', 'service')][string]$Auth = 'none'
    )
    $token = ''
    switch ($Auth) {
        'human' {
            $token = $humanToken
            if ([string]::IsNullOrEmpty($token)) {
                Add-Result $Service $Check SKIP 0 '' 'no human token available (set BSYSTEM_HUMAN_TOKEN)'
                return
            }
        }
        'service' {
            $token = $serviceToken
            if ([string]::IsNullOrEmpty($token)) {
                Add-Result $Service $Check SKIP 0 '' 'no service token available (set BSYSTEM_SERVICE_TOKEN); the machine API is not reachable with a human token by design'
                return
            }
        }
    }

    # The correlation id is sent so a red row can be found in the platform's
    # logs. It is not a secret and appears in the report deliberately.
    $requestId = 'smoke-{0}-{1}' -f (Get-Date -Format 'HHmmss'), (Get-Random -Maximum 99999)
    $headers = @{ 'X-Request-ID' = $requestId }
    if ($token) { $headers['Authorization'] = "Bearer $token" }

    $watch = [System.Diagnostics.Stopwatch]::StartNew()
    $status = 0
    try {
        $response = Invoke-WebRequest -Uri $Url -Headers $headers -TimeoutSec $TimeoutSeconds `
            -UseBasicParsing -SkipHttpErrorCheck:$false -ErrorAction Stop
        $status = [int]$response.StatusCode
    } catch [System.Net.WebException] {
        if ($_.Exception.Response) { $status = [int]$_.Exception.Response.StatusCode }
    } catch {
        # PowerShell 6+ surfaces HTTP errors as HttpResponseException.
        if ($_.Exception.PSObject.Properties.Name -contains 'Response' -and $_.Exception.Response) {
            $status = [int]$_.Exception.Response.StatusCode
        }
    }
    $watch.Stop()
    $duration = [int]$watch.ElapsedMilliseconds

    if ($status -eq 0) {
        Add-Result $Service $Check FAIL $duration $requestId "no response from $Url within ${TimeoutSeconds}s"
        return
    }
    if ($Expected -contains $status) {
        Add-Result $Service $Check PASS $duration $requestId "HTTP $status"
    } else {
        Add-Result $Service $Check FAIL $duration $requestId "HTTP $status, expected one of: $($Expected -join ', ')"
    }
}

function Get-Body {
    param([string]$Url)
    try {
        return (Invoke-WebRequest -Uri $Url -TimeoutSec $TimeoutSeconds -UseBasicParsing -ErrorAction Stop).Content
    } catch {
        return ''
    }
}

function Get-RepositoryCommit {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath (Join-Path $Path '.git'))) { return 'unavailable' }
    try {
        $sha = & git -C $Path rev-parse HEAD 2>$null
        if ($LASTEXITCODE -eq 0 -and $sha) { return $sha.Trim() }
    } catch { }
    return 'unknown'
}

Write-Host 'BSYSTEM stage smoke'
Write-Host "core=$CoreUrl hub=$HubUrl"
if (-not $humanToken) { Write-Host '  (no human token: authenticated checks will be skipped)' -ForegroundColor DarkGray }
if (-not $serviceToken) { Write-Host '  (no service token: machine API checks will be skipped)' -ForegroundColor DarkGray }

Write-Host ''; Write-Host '== Operational endpoints'
Invoke-Check 'integration-core' 'GET /health'  "$CoreUrl/health"  @(200)
Invoke-Check 'integration-core' 'GET /readyz'  "$CoreUrl/readyz"  @(200)
Invoke-Check 'integration-core' 'GET /metrics' "$CoreUrl/metrics" @(200)
Invoke-Check 'hub'              'GET /healthz' "$HubUrl/healthz"  @(200)

Write-Host ''; Write-Host '== Identity'
if ($AuthentikUrl) {
    Invoke-Check 'authentik' 'OIDC discovery' "$AuthentikUrl/.well-known/openid-configuration" @(200)
} else {
    Add-Result 'authentik' 'OIDC discovery' SKIP 0 '' 'AUTHENTIK_URL is not configured'
}

Write-Host ''; Write-Host '== Authorization boundary'
# These need no token: an unauthenticated caller must be rejected. A 200 here
# would be worth stopping the acceptance for.
Invoke-Check 'integration-core' 'GET /api/v1/me rejects anonymous'      "$CoreUrl/api/v1/me"                      @(401)
Invoke-Check 'integration-core' 'GET /api/v1/clients rejects anonymous' "$CoreUrl/api/v1/clients"                 @(401)
Invoke-Check 'integration-core' 'machine API rejects anonymous'         "$CoreUrl/api/service/v1/adapters/health" @(401)

Write-Host ''; Write-Host '== Normalized human API'
Invoke-Check 'integration-core' 'GET /api/v1/me'            "$CoreUrl/api/v1/me"            @(200)      human
Invoke-Check 'integration-core' 'GET /api/v1/modules'       "$CoreUrl/api/v1/modules"       @(200)      human
# 503 is accepted here, not tolerated: an adapter-backed collection answers 503
# when its source system is not configured, which is a legitimate stage state.
# Accepting only 200 and 403 would report a correctly behaving platform as
# broken whenever an integration is deliberately left out.
Invoke-Check 'integration-core' 'GET /api/v1/clients'       "$CoreUrl/api/v1/clients"       @(200, 403, 503) human
Invoke-Check 'integration-core' 'GET /api/v1/projects'      "$CoreUrl/api/v1/projects"      @(200, 403, 503) human
Invoke-Check 'integration-core' 'GET /api/v1/documents'     "$CoreUrl/api/v1/documents"     @(200, 403, 503) human
Invoke-Check 'integration-core' 'GET /api/v1/notifications' "$CoreUrl/api/v1/notifications" @(200)      human
Invoke-Check 'integration-core' 'GET /api/v1/search'        "$CoreUrl/api/v1/search?q=test" @(200)      human

Write-Host ''; Write-Host '== Machine API'
Invoke-Check 'integration-core' 'GET /api/service/v1/whoami'          "$CoreUrl/api/service/v1/whoami"          @(200) service
Invoke-Check 'integration-core' 'GET /api/service/v1/adapters'        "$CoreUrl/api/service/v1/adapters"        @(200) service
Invoke-Check 'integration-core' 'GET /api/service/v1/adapters/health' "$CoreUrl/api/service/v1/adapters/health" @(200) service

Write-Host ''; Write-Host "== Dependencies, through the platform's own readiness"
# PostgreSQL and NATS are on internal networks and are not probed directly:
# reaching them from outside would mean the topology is wrong.
$readiness = Get-Body "$CoreUrl/readyz"
if (-not $readiness) {
    Add-Result 'postgresql' 'reachable through /readyz' FAIL 0 '' 'no readiness response'
    Add-Result 'nats' 'reachable through /readyz' FAIL 0 '' 'no readiness response'
} else {
    if ($readiness -match '"database"\s*:\s*"ok"') {
        Add-Result 'postgresql' 'reachable through /readyz' PASS 0 '' 'the Core reports the database as ok'
    } else {
        Add-Result 'postgresql' 'reachable through /readyz' FAIL 0 '' 'the Core does not report the database as ok'
    }
    if ($readiness -match '"nats"\s*:\s*"ok"') {
        Add-Result 'nats' 'reachable through /readyz' PASS 0 '' 'the Core reports NATS as ok'
    } elseif ($readiness -match 'degraded') {
        Add-Result 'nats' 'reachable through /readyz' SKIP 0 '' 'NATS is degraded or not configured; the platform serves without it and drops events'
    } else {
        Add-Result 'nats' 'reachable through /readyz' FAIL 0 '' 'unrecognized readiness payload'
    }
}

$metrics = Get-Body "$CoreUrl/metrics"
if (-not $metrics) {
    Add-Result 'integration-core' 'build metadata exposed' FAIL 0 '' 'no metrics response'
} elseif ($metrics -match 'bsystem_build_info') {
    Add-Result 'integration-core' 'build metadata exposed' PASS 0 '' 'bsystem_build_info is present'
} else {
    Add-Result 'integration-core' 'build metadata exposed' FAIL 0 '' 'bsystem_build_info is absent; the running commit cannot be identified'
}

Write-Host ''; Write-Host '== Owner-blocked'
Add-Result 'platform' 'customer isolation with real ownership' BLOCKED 0 '' 'requires the authoritative customer ownership mapping; see TENANT-ISOLATION-MATRIX.md'
Add-Result 'platform' 'SLA timing' BLOCKED 0 '' 'requires response and resolution targets per severity'

$finishedAt = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$passed = @($results | Where-Object result -eq 'PASS').Count
$failed = @($results | Where-Object result -eq 'FAIL').Count
$skipped = @($results | Where-Object result -eq 'SKIP').Count
$blocked = @($results | Where-Object result -eq 'BLOCKED').Count

if (-not (Test-Path -LiteralPath $ArtifactsDir)) { New-Item -ItemType Directory -Path $ArtifactsDir | Out-Null }

$report = [ordered]@{
    started_at = $startedAt
    finished_at = $finishedAt
    targets = [ordered]@{ core = $CoreUrl; hub = $HubUrl }
    commits = [ordered]@{
        'bsystem-deploy' = Get-RepositoryCommit '.'
        'bsystem-integration-core' = Get-RepositoryCommit '../bsystem-integration-core'
        'bsystem-hub' = Get-RepositoryCommit '../bsystem-hub'
        'bsystem-design-system' = Get-RepositoryCommit '../bsystem-design-system'
    }
    summary = [ordered]@{ pass = $passed; fail = $failed; skip = $skipped; blocked = $blocked }
    checks = $results
}
$report | ConvertTo-Json -Depth 5 | Set-Content -Path (Join-Path $ArtifactsDir 'stage-acceptance.json') -Encoding UTF8

$markdown = [System.Text.StringBuilder]::new()
[void]$markdown.AppendLine('# Stage acceptance').AppendLine()
[void]$markdown.AppendLine('| | |').AppendLine('| --- | --- |')
[void]$markdown.AppendLine("| Started | ``$startedAt`` |")
[void]$markdown.AppendLine("| Finished | ``$finishedAt`` |")
[void]$markdown.AppendLine("| Core | ``$CoreUrl`` |")
[void]$markdown.AppendLine("| HUB | ``$HubUrl`` |").AppendLine()
[void]$markdown.AppendLine('## Commits').AppendLine()
[void]$markdown.AppendLine('| Repository | Commit |').AppendLine('| --- | --- |')
foreach ($entry in $report.commits.GetEnumerator()) {
    [void]$markdown.AppendLine("| ``$($entry.Key)`` | ``$($entry.Value)`` |")
}
[void]$markdown.AppendLine().AppendLine('## Result').AppendLine()
[void]$markdown.AppendLine("**$passed passed, $failed failed, $skipped skipped, $blocked blocked.**").AppendLine()
[void]$markdown.AppendLine('| Service | Check | Result | ms | Request ID | Note |')
[void]$markdown.AppendLine('| --- | --- | --- | --- | --- | --- |')
foreach ($row in $results) {
    [void]$markdown.AppendLine("| $($row.service) | $($row.check) | $($row.result) | $($row.duration_ms) | ``$($row.request_id)`` | $($row.message) |")
}
[void]$markdown.AppendLine().AppendLine('SKIP means a check could not run, usually for want of a token; it is not a pass.')
[void]$markdown.AppendLine('BLOCKED means the check needs something only the owner can supply.').AppendLine()
[void]$markdown.AppendLine('No credential, token or upstream payload appears in this report.')
$markdown.ToString() | Set-Content -Path (Join-Path $ArtifactsDir 'stage-acceptance.md') -Encoding UTF8

Write-Host ''
Write-Host "report: $(Join-Path $ArtifactsDir 'stage-acceptance.json')"
Write-Host "report: $(Join-Path $ArtifactsDir 'stage-acceptance.md')"
Write-Host "$passed passed, $failed failed, $skipped skipped, $blocked blocked"

if ($failed -gt 0) { exit 1 }
exit 0
