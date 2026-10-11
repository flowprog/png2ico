param([switch]$CLI)

$ErrorActionPreference = 'Stop'
Push-Location -LiteralPath $PSScriptRoot
try {
    $buildFlags = '-s -w'
    $outputPath = 'bin/png2ico.exe'
    $targetPackage = '.'
    if ($CLI) {
        $outputPath = 'bin/png2ico-cli.exe'
        $targetPackage = './cmd/png2ico-cli'
    } else {
        $buildFlags += ' -H=windowsgui -X github.com/egoist/mygo.production=1'
    }
    go build -trimpath -tags mygo_no_inspector -ldflags $buildFlags -o $outputPath $targetPackage
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed with exit code $LASTEXITCODE"
    }
    Write-Output (Join-Path $PSScriptRoot $outputPath)
} finally {
    Pop-Location
}
