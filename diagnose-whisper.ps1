Write-Host "==== Diagnóstico do Ambiente Python para CompressVideo ===="
Write-Host ""

# Verificar versão do PowerShell
$PSVersionTable.PSVersion
Write-Host ""

# Verificar Python no PATH
Write-Host "=== Verificando Python no PATH ==="
$pythonCommands = @("python", "python3")
$foundPython = $false

foreach ($cmd in $pythonCommands) {
    try {
        $version = & $cmd --version 2>&1
        Write-Host "✅ $cmd encontrado: $version"
        $path = (Get-Command $cmd).Source
        Write-Host "   Localização: $path"
        $foundPython = $true
        
        # Verificar pip
        Write-Host "   Verificando pip..."
        try {
            $pipVersion = & $cmd -m pip --version 2>&1
            Write-Host "   ✅ pip encontrado: $pipVersion"
        }
        catch {
            Write-Host "   ❌ pip não encontrado ou não funcionando"
        }
        
        # Verificar módulos
        Write-Host "   Verificando módulos whisper..."
        try {
            & $cmd -c "import whisper_ctranslate2; print('✅ whisper_ctranslate2 encontrado')" 2>&1
        }
        catch {
            Write-Host "   ❌ Erro ao importar whisper_ctranslate2: $_"
        }
        
        try {
            & $cmd -c "import whisper; print('✅ openai-whisper encontrado')" 2>&1
        }
        catch {
            Write-Host "   ❌ Erro ao importar openai-whisper: $_"
        }
        
        Write-Host ""
    }
    catch {
        Write-Host "❌ $cmd não encontrado ou não funcionando"
    }
}

if (-not $foundPython) {
    Write-Host "Nenhuma instalação de Python encontrada no PATH"
}

# Verificar caminhos comuns de instalação do Python
Write-Host "=== Verificando instalações Python fora do PATH ==="
$commonPaths = @(
    "C:\Python*\python.exe",
    "C:\Program Files\Python*\python.exe",
    "C:\Program Files (x86)\Python*\python.exe",
    "$env:LOCALAPPDATA\Programs\Python\Python*\python.exe",
    "$env:APPDATA\Python\Python*\python.exe"
)

$foundAdditional = $false

foreach ($pattern in $commonPaths) {
    $pythonInstalls = Get-Item $pattern -ErrorAction SilentlyContinue
    foreach ($python in $pythonInstalls) {
        $foundAdditional = $true
        Write-Host "Instalação encontrada: $($python.FullName)"
        
        try {
            $version = & $python.FullName --version 2>&1
            Write-Host "✅ Versão: $version"
            
            # Verificar whisper_ctranslate2
            try {
                & $python.FullName -c "import whisper_ctranslate2; print('✅ whisper_ctranslate2 encontrado')" 2>&1
            }
            catch {
                Write-Host "❌ Erro ao importar whisper_ctranslate2"
            }
        }
        catch {
            Write-Host "❌ Não foi possível executar este Python"
        }
        
        Write-Host ""
    }
}

if (-not $foundAdditional) {
    Write-Host "Nenhuma instalação adicional de Python encontrada"
}

# Verificar registro do Windows
Write-Host "=== Verificando registro do Windows para instalações Python ==="
try {
    $regPaths = Get-ChildItem "HKLM:\SOFTWARE\Python\PythonCore" -ErrorAction SilentlyContinue | 
                Select-Object -ExpandProperty Name
    
    if ($regPaths) {
        foreach ($path in $regPaths) {
            $version = $path.Split('\')[-1]
            Write-Host "Python $version encontrado no registro"
            
            try {
                $installPath = (Get-ItemProperty "HKLM:\SOFTWARE\Python\PythonCore\$version\InstallPath" -ErrorAction SilentlyContinue).'(default)'
                if ($installPath) {
                    $fullPath = Join-Path $installPath "python.exe"
                    Write-Host "Caminho: $fullPath"
                    
                    if (Test-Path $fullPath) {
                        Write-Host "✅ Executável existe"
                    } else {
                        Write-Host "❌ Executável não encontrado"
                    }
                }
            }
            catch {
                Write-Host "❌ Não foi possível obter caminho de instalação"
            }
        }
    } else {
        Write-Host "Nenhuma instalação de Python encontrada no registro"
    }
}
catch {
    Write-Host "Erro ao verificar registro: $_"
}

# Informações de ambiente
Write-Host ""
Write-Host "=== Variáveis de ambiente relevantes ==="
Write-Host "PATH: $env:PATH"

# Sugestões finais
Write-Host ""
Write-Host "=== Sugestões ==="
Write-Host "1. Certifique-se de que o Python está no PATH"
Write-Host "2. Tente reinstalar whisper-ctranslate2: pip install -U whisper-ctranslate2"
Write-Host "3. Execute o compressvideo como administrador"
Write-Host "4. Tente criar um ambiente virtual Python dedicado para o compressvideo" 