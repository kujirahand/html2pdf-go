# Install the latest (or a given) html2pdf-go release for Windows.
#
#   irm https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.ps1 | iex
#
# Environment:
#   HTML2PDF_VERSION      version tag to install (default: latest)
#   HTML2PDF_INSTALL_DIR  install directory (default: %LOCALAPPDATA%\Programs\html2pdf-go)
param(
    [string]$Version = $env:HTML2PDF_VERSION
)

$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$repo = 'kujirahand/html2pdf-go'

if (-not $Version) { $Version = 'latest' }

switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    default { throw "install.ps1: unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
}

if ($Version -eq 'latest') {
    $tag = $null
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -UseBasicParsing
        $tag = $release.tag_name
    } catch {
        # The API is rate-limited for unauthenticated callers; fall back to
        # the /releases/latest redirect, which has no such quota.
        try {
            $resp = Invoke-WebRequest -Uri "https://github.com/$repo/releases/latest" -UseBasicParsing
            $final = $resp.BaseResponse.ResponseUri.AbsoluteUri
            if ($final -match '/tag/(.+)$') { $tag = $Matches[1] }
        } catch { }
    }
    if (-not $tag) { throw 'install.ps1: could not determine the latest release tag' }
} else {
    $tag = $Version
}

# A tag may contain '/' (e.g. release/v1.0.0); build-release.sh replaces
# it with '-' in archive names, while the download URL keeps the raw tag.
$fileTag = $tag -replace '/', '-'
$archive = "html2pdf-go_${fileTag}_windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/$tag"

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("html2pdf-go-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
    Write-Host "install.ps1: downloading $archive"
    $zipPath = Join-Path $tmp $archive
    $sumPath = Join-Path $tmp 'checksums.txt'
    Invoke-WebRequest -Uri "$base/$archive" -OutFile $zipPath -UseBasicParsing
    Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sumPath -UseBasicParsing

    $expected = $null
    foreach ($line in Get-Content $sumPath) {
        $parts = $line.Trim() -split '\s+', 2
        if ($parts.Count -eq 2 -and $parts[1].TrimStart('*') -eq $archive) {
            $expected = $parts[0]
            break
        }
    }
    if (-not $expected) { throw "install.ps1: $archive not found in checksums.txt" }

    $actual = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash
    if ($actual.ToLower() -ne $expected.ToLower()) {
        throw "install.ps1: checksum mismatch for $archive (expected $expected, got $actual); nothing was installed"
    }
    Write-Host 'install.ps1: checksum OK'

    $extract = Join-Path $tmp 'extract'
    Expand-Archive -Path $zipPath -DestinationPath $extract -Force

    $dir = $env:HTML2PDF_INSTALL_DIR
    if (-not $dir) { $dir = Join-Path $env:LOCALAPPDATA 'Programs\html2pdf-go' }
    New-Item -ItemType Directory -Path $dir -Force | Out-Null

    foreach ($name in 'html2pdf.exe', 'md2pdf.exe') {
        $src = Get-ChildItem -Path $extract -Recurse -Filter $name | Select-Object -First 1
        if (-not $src) { throw "install.ps1: $name not found in $archive" }
        Copy-Item -Path $src.FullName -Destination (Join-Path $dir $name) -Force
    }

    Write-Host "install.ps1: installed html2pdf.exe and md2pdf.exe to $dir"
    & (Join-Path $dir 'html2pdf.exe') -version

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $entries = @()
    if ($userPath) { $entries = $userPath -split ';' | ForEach-Object { $_.TrimEnd('\') } }
    if ($entries -notcontains $dir.TrimEnd('\')) {
        Write-Host "install.ps1: $dir is not in your PATH. Add it to the user PATH with:"
        Write-Host "  [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$dir', 'User')"
        Write-Host '  (then restart the terminal)'
    }
} finally {
    Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
