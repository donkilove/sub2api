[CmdletBinding()]
param(
  [ValidateSet('quick', 'full', 'release')]
  [string]$Mode = 'quick',

  [string[]]$FrontendTest = @(),
  [string[]]$BackendPackage = @(),
  [string]$ImageTag = '',

  [switch]$Force,
  [switch]$SkipFullChecks,
  [switch]$SkipDocker,
  [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$frontendDir = Join-Path $repoRoot 'frontend'
$backendDir = Join-Path $repoRoot 'backend'

$criticalFrontendTests = @(
  'src/views/auth/__tests__/LinuxDoCallbackView.spec.ts',
  'src/views/auth/__tests__/WechatCallbackView.spec.ts',
  'src/views/user/__tests__/PaymentView.spec.ts',
  'src/views/user/__tests__/PaymentResultView.spec.ts',
  'src/components/user/profile/__tests__/ProfileInfoCard.spec.ts',
  'src/views/admin/__tests__/SettingsView.spec.ts'
)

function Write-GateInfo {
  param([string]$Message)
  Write-Host "[gate] $Message"
}

function Invoke-GateStep {
  param(
    [string]$Name,
    [string]$WorkingDirectory,
    [string]$Command,
    [string[]]$Arguments = @()
  )

  $display = "$Command $($Arguments -join ' ')".Trim()
  Write-GateInfo "$Name"
  Write-Host "       cwd: $WorkingDirectory"
  Write-Host "       cmd: $display"

  if ($DryRun) {
    return
  }

  Push-Location $WorkingDirectory
  try {
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
      throw "Gate step failed: $Name (exit code $LASTEXITCODE)"
    }
  } finally {
    Pop-Location
  }
}

function Get-GitOutputLines {
  param([string[]]$Arguments)

  Push-Location $repoRoot
  try {
    $previousErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    $output = & git @Arguments 2>$null
    $ErrorActionPreference = $previousErrorActionPreference
    if ($LASTEXITCODE -ne 0) {
      return @()
    }
    return @($output | Where-Object { $_ })
  } finally {
    if (Get-Variable -Name previousErrorActionPreference -Scope Local -ErrorAction SilentlyContinue) {
      $ErrorActionPreference = $previousErrorActionPreference
    }
    Pop-Location
  }
}

function Get-ChangedFilesForQuickGate {
  $files = @()
  $files += Get-GitOutputLines @('diff', '--name-only')
  $files += Get-GitOutputLines @('diff', '--cached', '--name-only')
  $files += Get-GitOutputLines @('ls-files', '--others', '--exclude-standard')

  if ($files.Count -eq 0) {
    $files += Get-GitOutputLines @('diff', '--name-only', 'HEAD~1..HEAD')
  }

  return @(
    $files |
      Where-Object { ($_ -replace '\\', '/') -notlike '.superpowers/*' } |
      Sort-Object -Unique
  )
}

function Test-PathPrefix {
  param(
    [string[]]$Files,
    [string[]]$Prefixes
  )

  foreach ($file in $Files) {
    $normalized = $file -replace '\\', '/'
    foreach ($prefix in $Prefixes) {
      if ($normalized.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $true
      }
    }
  }
  return $false
}

function Invoke-FrontendQuickGate {
  $tests = if ($FrontendTest.Count -gt 0) { $FrontendTest } else { $criticalFrontendTests }

  Invoke-GateStep `
    -Name 'frontend typecheck' `
    -WorkingDirectory $repoRoot `
    -Command 'corepack' `
    -Arguments @('pnpm', '--dir', 'frontend', 'run', 'typecheck')

  Invoke-GateStep `
    -Name 'frontend quick tests' `
    -WorkingDirectory $repoRoot `
    -Command 'corepack' `
    -Arguments (@('pnpm', '--dir', 'frontend', 'exec', 'vitest', 'run') + $tests)
}

function Invoke-BackendQuickGate {
  $packages = if ($BackendPackage.Count -gt 0) { $BackendPackage } else { @('./...') }

  Invoke-GateStep `
    -Name 'backend quick tests' `
    -WorkingDirectory $backendDir `
    -Command 'go' `
    -Arguments (@('test') + $packages)
}

function Invoke-FullGate {
  Invoke-GateStep `
    -Name 'frontend lint' `
    -WorkingDirectory $repoRoot `
    -Command 'corepack' `
    -Arguments @('pnpm', '--dir', 'frontend', 'run', 'lint:check')

  Invoke-GateStep `
    -Name 'frontend typecheck' `
    -WorkingDirectory $repoRoot `
    -Command 'corepack' `
    -Arguments @('pnpm', '--dir', 'frontend', 'run', 'typecheck')

  Invoke-GateStep `
    -Name 'frontend full tests' `
    -WorkingDirectory $repoRoot `
    -Command 'corepack' `
    -Arguments @('pnpm', '--dir', 'frontend', 'exec', 'vitest', 'run')

  Invoke-GateStep `
    -Name 'backend full tests' `
    -WorkingDirectory $backendDir `
    -Command 'go' `
    -Arguments @('test', './...')

  Invoke-GateStep `
    -Name 'backend lint' `
    -WorkingDirectory $backendDir `
    -Command 'golangci-lint' `
    -Arguments @('run', './...', '--timeout=30m')
}

function Invoke-ReleaseGate {
  if (-not $SkipFullChecks) {
    Invoke-FullGate
  } else {
    Write-GateInfo 'skip full checks: caller confirmed this commit already passed full gate'
  }

  Invoke-GateStep `
    -Name 'frontend production build' `
    -WorkingDirectory $repoRoot `
    -Command 'corepack' `
    -Arguments @('pnpm', '--dir', 'frontend', 'run', 'build')

  if ($SkipDocker) {
    Write-GateInfo 'skip Docker build'
    return
  }

  $commit = (Get-GitOutputLines @('rev-parse', '--short=12', 'HEAD'))[0]
  $date = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
  $tag = $ImageTag
  if ([string]::IsNullOrWhiteSpace($tag)) {
    $stamp = (Get-Date).ToUniversalTime().ToString('yyyyMMdd')
    $tag = "donki/sub2api:manual-$stamp-$commit"
  }

  Invoke-GateStep `
    -Name "Docker build $tag" `
    -WorkingDirectory $repoRoot `
    -Command 'docker' `
    -Arguments @('build', '--build-arg', "COMMIT=$commit", '--build-arg', "DATE=$date", '-t', $tag, '.')
}

Write-GateInfo "mode=$Mode repo=$repoRoot"

switch ($Mode) {
  'quick' {
    [string[]]$changedFiles = @(Get-ChangedFilesForQuickGate)
    $frontendTouched = $Force -or $FrontendTest.Count -gt 0 -or (Test-PathPrefix $changedFiles @('frontend/', 'docs/legal/'))
    $backendTouched = $Force -or $BackendPackage.Count -gt 0 -or (Test-PathPrefix $changedFiles @('backend/'))

    if ($changedFiles.Count -gt 0) {
      Write-GateInfo "quick changed files: $($changedFiles -join ', ')"
    } else {
      Write-GateInfo 'quick found no Git changes; use -Force to run both frontend and backend quick checks'
    }

    if ($frontendTouched) {
      Invoke-FrontendQuickGate
    } else {
      Write-GateInfo 'skip frontend quick: no frontend-related changes detected'
    }

    if ($backendTouched) {
      Invoke-BackendQuickGate
    } else {
      Write-GateInfo 'skip backend quick: no backend-related changes detected'
    }
  }
  'full' {
    Invoke-FullGate
  }
  'release' {
    Invoke-ReleaseGate
  }
}

Write-GateInfo 'done'
