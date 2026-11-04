# Install bibx from GitHub Releases.
#
#   irm https://raw.githubusercontent.com/coreofscience/go-bibx/main/installers/install.ps1 | iex
#
# Environment variables:
#   BIBX_VERSION      release tag to install, e.g. v0.7.0 (default: latest)
#   BIBX_INSTALL_DIR  where to put bibx.exe (default: $env:LOCALAPPDATA\Programs\bibx)

function Install-Bibx {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'
    $repo = 'coreofscience/go-bibx'

    if ($PSVersionTable.PSVersion.Major -lt 6) {
        [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    }

    $arch = $env:PROCESSOR_ARCHITEW6432
    if (-not $arch) { $arch = $env:PROCESSOR_ARCHITECTURE }
    switch ($arch) {
        'AMD64' { }
        'ARM64' { Write-Warning 'No native ARM64 build; installing the x86_64 build, which runs under emulation.' }
        default { throw "Unsupported architecture '$arch'." }
    }

    $version = $env:BIBX_VERSION
    if (-not $version) {
        # Follow the releases/latest redirect instead of the rate-limited API.
        $request = [Net.HttpWebRequest]::Create("https://github.com/$repo/releases/latest")
        $request.AllowAutoRedirect = $false
        $response = $request.GetResponse()
        try { $location = $response.Headers['Location'] } finally { $response.Close() }
        $version = ($location -split '/')[-1]
        if (-not $version.StartsWith('v')) { throw "Could not determine the latest release (got '$location')." }
    }
    if (-not $version.StartsWith('v')) { $version = "v$version" }

    $installDir = $env:BIBX_INSTALL_DIR
    if (-not $installDir) { $installDir = Join-Path $env:LOCALAPPDATA 'Programs\bibx' }

    $archive = 'go-bibx_Windows_x86_64.zip'
    $checksums = "go-bibx_$($version.Substring(1))_checksums.txt"
    $base = "https://github.com/$repo/releases/download/$version"

    $tmp = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $tmp | Out-Null
    try {
        Write-Host "Downloading bibx $version for Windows/x86_64"
        Invoke-WebRequest -UseBasicParsing -Uri "$base/$archive" -OutFile (Join-Path $tmp $archive)
        Invoke-WebRequest -UseBasicParsing -Uri "$base/$checksums" -OutFile (Join-Path $tmp $checksums)

        $line = Get-Content (Join-Path $tmp $checksums) | Where-Object { ($_ -split '\s+')[1] -eq $archive }
        if (-not $line) { throw "No checksum for $archive in $checksums." }
        $expected = ($line -split '\s+')[0].ToLower()
        $actual = (Get-FileHash -Algorithm SHA256 -Path (Join-Path $tmp $archive)).Hash.ToLower()
        if ($actual -ne $expected) { throw "Checksum mismatch for $archive (expected $expected, got $actual)." }

        Expand-Archive -Path (Join-Path $tmp $archive) -DestinationPath (Join-Path $tmp 'out') -Force
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        # Releases up to v0.7.0 named the binary go-bibx.exe.
        $binary = Join-Path $tmp 'out\bibx.exe'
        if (-not (Test-Path $binary)) { $binary = Join-Path $tmp 'out\go-bibx.exe' }
        if (-not (Test-Path $binary)) { throw "No bibx binary in $archive." }
        Copy-Item -Path $binary -Destination (Join-Path $installDir 'bibx.exe') -Force
    }
    finally {
        Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }

    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $entries = @($userPath -split ';' | Where-Object { $_ })
    if ($entries -notcontains $installDir) {
        [Environment]::SetEnvironmentVariable('Path', (($entries + $installDir) -join ';'), 'User')
        Write-Host "Added $installDir to your user PATH; open a new terminal to pick it up."
    }
    if (($env:Path -split ';') -notcontains $installDir) {
        $env:Path = "$env:Path;$installDir"
    }

    $installed = & (Join-Path $installDir 'bibx.exe') --version
    Write-Host "Installed $installed to $installDir\bibx.exe"
}

Install-Bibx
