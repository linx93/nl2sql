$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$patterns = @("*.go", "*.md", "*.yaml", "*.yml", "*.sql", "*.ps1", "*.sh")
$suspectPattern = "缃|锟|鈥|闂|ï¿½|Ã|Â"

$files = Get-ChildItem -Path $repoRoot -Recurse -File | Where-Object {
    foreach ($pattern in $patterns) {
        if ($_.Name -like $pattern) { return $true }
    }
    return $false
}

foreach ($file in $files) {
    $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
    try {
        $content = [System.Text.Encoding]::UTF8.GetString($bytes)
    } catch {
        throw "文件不是 UTF-8: $($file.FullName)"
    }

    if ($content.Contains([char]0xFFFD)) {
        throw "文件包含 UTF-8 替换字符，疑似编码损坏: $($file.FullName)"
    }

    if ($content -match $suspectPattern) {
        throw "文件包含疑似中文乱码片段，请人工检查: $($file.FullName)"
    }
}

Write-Host "encoding check passed"
