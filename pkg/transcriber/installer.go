package transcriber

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cccarv82/compressvideo/pkg/util"
)

// WhisperInstaller é responsável pela instalação do Whisper
type WhisperInstaller struct {
	Logger *util.Logger
}

// NewWhisperInstaller cria uma nova instância do instalador do Whisper
func NewWhisperInstaller(logger *util.Logger) *WhisperInstaller {
	return &WhisperInstaller{
		Logger: logger,
	}
}

// CheckWhisperInstallation verifica se o Whisper está instalado
func (wi *WhisperInstaller) CheckWhisperInstallation() (bool, error) {
	whisperPath, err := findWhisper()
	if err != nil || whisperPath == "" {
		return false, nil
	}
	return true, nil
}

// InstallWhisper instala o Whisper
func (wi *WhisperInstaller) InstallWhisper(progress *util.ProgressTracker) error {
	wi.Logger.Info("Instalando Whisper...")
	
	// Primeiro tentar usar a instalação via pip (mais fácil)
	if canUsePython() {
		pythonCmd, err := findPythonCommand()
		if err != nil {
			wi.Logger.Warning("Não foi possível encontrar Python: %v", err)
		} else {
			wi.Logger.Info("Python encontrado: %s", pythonCmd)
			
			// Tentar instalar whisper-ctranslate2 (mais fácil no Windows)
			if util.IsWindows() {
				wi.Logger.Info("Tentando instalar whisper-ctranslate2 (recomendado para Windows)...")
				if whisperPath, err := installWhisperCtranslate2(pythonCmd, wi.Logger); err == nil {
					wi.Logger.Success("whisper-ctranslate2 instalado com sucesso em: %s", whisperPath)
					return nil
				} else {
					wi.Logger.Warning("Falha ao instalar whisper-ctranslate2: %v", err)
				}
			}
			
			// Instalar a versão Python do Whisper
			whisperPath, err := installWhisperPython(pythonCmd, wi.Logger)
			if err == nil {
				wi.Logger.Success("Whisper instalado com sucesso em: %s", whisperPath)
				return nil
			}
			wi.Logger.Error("Falha ao instalar Whisper (Python): %v", err)
		}
	}
	
	// Se não conseguiu com Python, tentar C++
	wi.Logger.Info("Tentando instalar versão C++...")
	whisperPath, err := installWhisperCPP(wi.Logger)
	if err != nil {
		// Se falhar no Windows, oferecer instruções alternativas
		if util.IsWindows() {
			wi.Logger.Error("Não foi possível instalar o Whisper automaticamente no Windows.")
			wi.Logger.Info("Por favor, execute o seguinte comando no PowerShell ou prompt de comando:")
			wi.Logger.Info("pip install -U whisper-ctranslate2")
			wi.Logger.Info("Após a instalação, crie um arquivo whisper.bat em um diretório do PATH com o conteúdo:")
			wi.Logger.Info("@echo off")
			wi.Logger.Info("whisper-ctranslate2 %*")
			return fmt.Errorf("falha ao instalar Whisper no Windows. Siga as instruções manuais acima")
		}
		return fmt.Errorf("falha ao instalar Whisper: %w", err)
	}
	
	wi.Logger.Success("Whisper instalado com sucesso em: %s", whisperPath)
	return nil
}

// InstallWhisperCtranslate2 instala apenas a versão mais leve do Whisper (whisper-ctranslate2)
func (wi *WhisperInstaller) InstallWhisperCtranslate2(progress *util.ProgressTracker) error {
	wi.Logger.Info("Instalando whisper-ctranslate2...")
	
	// Procurar por Python
	pythonCmd, err := findPythonCommand()
	if err != nil {
		if util.IsWindows() {
			// No Windows, verificar se o Python está instalado mas não está no PATH
			wi.Logger.Warning("Python não encontrado no PATH. Verificando caminhos comuns...")
			
			// Instruções para o usuário
			wi.Logger.Info("Verifique se o Python está instalado corretamente")
			wi.Logger.Info("Certifique-se de que a opção 'Add Python to PATH' foi selecionada durante a instalação")
			wi.Logger.Info("Como alternativa, execute manualmente: pip install -U whisper-ctranslate2")
		}
		return fmt.Errorf("Python não encontrado no sistema: %w", err)
	}
	
	// Instalar whisper-ctranslate2
	whisperPath, err := installWhisperCtranslate2(pythonCmd, wi.Logger)
	if err != nil {
		return fmt.Errorf("falha ao instalar whisper-ctranslate2: %w", err)
	}
	
	wi.Logger.Success("whisper-ctranslate2 instalado com sucesso em: %s", whisperPath)
	return nil
}

// findPythonCommand tenta encontrar o comando Python correto, considerando as peculiaridades do Windows
func findPythonCommand() (string, error) {
	var pythonPaths []string
	isWindows := util.IsWindows()
	
	// No Windows, verificar caminhos comuns primeiro
	if isWindows {
		// Obter variáveis de ambiente importantes
		userProfile := os.Getenv("USERPROFILE")
		programFiles := os.Getenv("ProgramFiles")
		programFilesX86 := os.Getenv("ProgramFiles(x86)")
		localAppData := os.Getenv("LOCALAPPDATA")
		
		// Caminhos comuns de instalação do Python no Windows
		commonPaths := []string{}
		
		// AppData/Local/Programs/Python
		if localAppData != "" {
			pythonPaths := findPythonInstalationsIn(filepath.Join(localAppData, "Programs", "Python"))
			commonPaths = append(commonPaths, pythonPaths...)
		}
		
		// UserProfile/AppData/Local/Programs/Python
		if userProfile != "" {
			pythonPaths := findPythonInstalationsIn(filepath.Join(userProfile, "AppData", "Local", "Programs", "Python"))
			commonPaths = append(commonPaths, pythonPaths...)
		}
		
		// Program Files
		if programFiles != "" {
			pythonPaths := findPythonInstalationsIn(programFiles)
			commonPaths = append(commonPaths, pythonPaths...)
		}
		
		// Program Files (x86)
		if programFilesX86 != "" {
			pythonPaths := findPythonInstalationsIn(programFilesX86)
			commonPaths = append(commonPaths, pythonPaths...)
		}
		
		// C:\Python*
		cPythonPaths := findPythonInstalationsIn("C:\\")
		commonPaths = append(commonPaths, cPythonPaths...)
		
		// Adicionar Windows Store Python
		if userProfile != "" {
			storePythonPaths := findPythonInstalationsIn(filepath.Join(userProfile, "AppData", "Local", "Microsoft", "WindowsApps"))
			commonPaths = append(commonPaths, storePythonPaths...)
		}
		
		// Adicionar os caminhos encontrados
		pythonPaths = append(pythonPaths, commonPaths...)
	}
	
	// Adicionar comandos padrão (python3, python) depois dos caminhos específicos
	pythonPaths = append(pythonPaths, "python3", "python")
	
	// Remover duplicatas mantendo a ordem
	seen := make(map[string]bool)
	var uniquePaths []string
	for _, path := range pythonPaths {
		if !seen[path] {
			seen[path] = true
			uniquePaths = append(uniquePaths, path)
		}
	}
	
	// Log para diagnóstico
	// fmt.Printf("Caminhos Python a verificar: %v\n", uniquePaths)
	
	// Verificar cada comando/caminho
	for _, cmdPath := range uniquePaths {
		foundPath, err := exec.LookPath(cmdPath)
		if err == nil {
			// Em sistemas Unix, qualquer Python encontrado é válido
			if !isWindows {
				return foundPath, nil
			}
			
			// No Windows, verificar se é um Python real (não um alias)
			// Executar python --version para testar
			cmd := exec.Command(foundPath, "--version")
			output, err := cmd.CombinedOutput()
			
			if err == nil && strings.Contains(strings.ToLower(string(output)), "python") {
				// Verificar também a capacidade de instalar pacotes (para confirmar que é uma instalação real)
				pipCmd := exec.Command(foundPath, "-m", "pip", "--version")
				pipOutput, pipErr := pipCmd.CombinedOutput()
				
				if pipErr == nil && strings.Contains(string(pipOutput), "pip") {
					// fmt.Printf("Python encontrado e válido: %s\n", foundPath)
					return foundPath, nil
				}
			}
		}
	}
	
	return "", fmt.Errorf("não foi possível encontrar um comando Python válido")
}

// findPythonInstalationsIn procura instalações do Python em um diretório base
func findPythonInstalationsIn(baseDir string) []string {
	var paths []string
	
	// Verificar se o diretório existe
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return paths
	}
	
	// Padrões para encontrar instalações Python
	patterns := []string{
		filepath.Join(baseDir, "Python*", "python.exe"),     // Python padrão
		filepath.Join(baseDir, "*", "Python*", "python.exe"), // Subdiretórios
		filepath.Join(baseDir, "python.exe"),                // Direto no diretório
	}
	
	// Procurar usando cada padrão
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			paths = append(paths, matches...)
		}
	}
	
	return paths
}

// canUsePython verifica se o Python está disponível no sistema
func canUsePython() bool {
	cmd, err := findPythonCommand()
	return err == nil && cmd != ""
}

// installWhisperCtranslate2 instala a versão ctranslate2 do Whisper (mais fácil no Windows)
func installWhisperCtranslate2(pythonCmd string, logger *util.Logger) (string, error) {
	logger.Info("Instalando whisper-ctranslate2...")
	
	// Verificar se pip está disponível
	pipCmd := filepath.Join(filepath.Dir(pythonCmd), "pip")
	if util.IsWindows() {
		pipCmd += ".exe"
	}
	
	if _, err := os.Stat(pipCmd); os.IsNotExist(err) {
		// Se pip não estiver junto com o python, usar o próprio python para chamar pip
		pipCmd = pythonCmd
		
		// Instalar whisper-ctranslate2
		cmd := exec.Command(pipCmd, "-m", "pip", "install", "-U", "whisper-ctranslate2")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("falha ao instalar via pip: %w\nOutput: %s", err, output)
		}
	} else {
		// Usar pip diretamente
		cmd := exec.Command(pipCmd, "install", "-U", "whisper-ctranslate2")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("falha ao instalar via pip: %w\nOutput: %s", err, output)
		}
	}
	
	// Criar wrapper
	wrapperPath, err := createWhisperCtranslate2Wrapper(pythonCmd, logger)
	if err != nil {
		return "", err
	}
	
	return wrapperPath, nil
}

// createWhisperCtranslate2Wrapper cria um arquivo batch para chamar whisper-ctranslate2
func createWhisperCtranslate2Wrapper(pythonCmd string, logger *util.Logger) (string, error) {
	logger.Info("Criando wrapper para whisper-ctranslate2...")
	
	// Determinar diretório para o wrapper
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar o diretório home: %w", err)
	}
	
	// Caminhos para diferentes sistemas operacionais
	var binDir string
	if util.IsWindows() {
		binDir = filepath.Join(homeDir, ".compressvideo", "bin")
	} else {
		binDir = filepath.Join(homeDir, ".local", "bin")
	}
	
	// Criar diretório se não existir
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório: %w", err)
	}
	
	// Caminho para o script wrapper
	wrapperPath := filepath.Join(binDir, "whisper")
	if util.IsWindows() {
		wrapperPath += ".bat"
	}
	
	// Verificar caminhos para o whisper-ctranslate2
	var whisperCtranslate2Path string
	var scriptPath string
	
	if util.IsWindows() {
		// Obter o diretório de scripts do Python (onde pip instala os executáveis)
		pythonDir := filepath.Dir(pythonCmd)
		scriptsDir := filepath.Join(pythonDir, "Scripts")
		
		// Procurar pelo executável whisper-ctranslate2.exe no diretório Scripts
		possiblePath := filepath.Join(scriptsDir, "whisper-ctranslate2.exe")
		if _, err := os.Stat(possiblePath); err == nil {
			whisperCtranslate2Path = possiblePath
			logger.Debug("Encontrado executável whisper-ctranslate2: %s", whisperCtranslate2Path)
		}
		
		// Se não encontrou o executável, verificar o script Python
		if whisperCtranslate2Path == "" {
			possiblePath = filepath.Join(scriptsDir, "whisper-ctranslate2-script.py")
			if _, err := os.Stat(possiblePath); err == nil {
				scriptPath = possiblePath
				logger.Debug("Encontrado script whisper-ctranslate2: %s", scriptPath)
			}
		}
	}
	
	// Conteúdo do script
	var content string
	if util.IsWindows() {
		// Melhorar o script batch para Windows
		if whisperCtranslate2Path != "" {
			// Se encontramos o executável direto, usar ele
			content = "@echo off\r\n" +
				"setlocal\r\n" +
				"set \"PATH=%PATH%;%~dp0\"\r\n" +
				"\"" + whisperCtranslate2Path + "\" %*\r\n"
		} else if scriptPath != "" {
			// Se encontramos o script Python, usar ele
			content = "@echo off\r\n" +
				"setlocal\r\n" +
				"set \"PATH=%PATH%;%~dp0\"\r\n" +
				"\"" + pythonCmd + "\" \"" + scriptPath + "\" %*\r\n"
		} else {
			// Usar o módulo diretamente como último recurso
			content = "@echo off\r\n" +
				"setlocal\r\n" +
				"set \"PATH=%PATH%;%~dp0\"\r\n" +
				"\"" + pythonCmd + "\" -m whisper_ctranslate2 %*\r\n"
		}
	} else {
		// Script para Linux/MacOS
		content = "#!/bin/sh\n" +
			"export PATH=\"$PATH:$(dirname \"$0\")\"\n" +
			pythonCmd + " -m whisper_ctranslate2 \"$@\"\n"
	}
	
	// Escrever o script
	if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
		return "", fmt.Errorf("falha ao criar wrapper: %w", err)
	}
	
	logger.Success("Wrapper criado com sucesso: %s", wrapperPath)
	
	// No Windows, verificar se o diretório está no PATH
	if util.IsWindows() {
		pathEnv := os.Getenv("PATH")
		if !strings.Contains(strings.ToLower(pathEnv), strings.ToLower(binDir)) {
			logger.Warning("O diretório %s não está no seu PATH", binDir)
			logger.Info("Para usar o comando 'whisper' globalmente, adicione este diretório ao PATH do sistema")
			logger.Info("Enquanto isso, você pode usar o caminho completo: %s", wrapperPath)
			
			// Tentar adicionar ao PATH temporariamente
			os.Setenv("PATH", binDir+string(os.PathListSeparator)+pathEnv)
		}
	}
	
	return wrapperPath, nil
}

// installWhisperPython instala o Whisper usando pip
func installWhisperPython(pythonCmd string, logger *util.Logger) (string, error) {
	logger.Info("Instalando Whisper usando Python...")
	
	// Verificar se pip está disponível
	pipCmd := filepath.Join(filepath.Dir(pythonCmd), "pip")
	if util.IsWindows() {
		pipCmd += ".exe"
	}
	
	// Instalar Whisper via pip
	logger.Info("Instalando pacote openai-whisper...")
	var cmd *exec.Cmd
	
	if _, err := os.Stat(pipCmd); os.IsNotExist(err) {
		// Se pip não estiver junto com o python, usar o próprio python para chamar pip
		cmd = exec.Command(pythonCmd, "-m", "pip", "install", "openai-whisper")
	} else {
		// Usar pip diretamente
		cmd = exec.Command(pipCmd, "install", "openai-whisper")
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("falha ao instalar via pip: %w\nOutput: %s", err, output)
	}
	
	// Criar um script wrapper para facilitar o uso
	wrapperPath, err := createWhisperWrapper(pythonCmd, logger)
	if err != nil {
		return "", err
	}
	
	return wrapperPath, nil
}

// createWhisperWrapper cria um script para facilitar o uso do Whisper
func createWhisperWrapper(pythonCmd string, logger *util.Logger) (string, error) {
	logger.Info("Criando wrapper para Whisper...")
	
	// Determinar diretório para o wrapper
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar o diretório home: %w", err)
	}
	
	binDir := filepath.Join(homeDir, ".local", "bin")
	if util.IsWindows() {
		binDir = filepath.Join(homeDir, "AppData", "Local", "CompressVideo", "bin")
	}
	
	// Criar diretório se não existir
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório: %w", err)
	}
	
	// Caminho para o script wrapper
	wrapperPath := filepath.Join(binDir, "whisper")
	if util.IsWindows() {
		wrapperPath += ".bat"
	}
	
	// Conteúdo do script
	var content string
	if util.IsWindows() {
		content = fmt.Sprintf("@echo off\n\"%s\" -m whisper %%*", pythonCmd)
	} else {
		content = fmt.Sprintf("#!/bin/sh\n%s -m whisper \"$@\"", pythonCmd)
	}
	
	// Escrever o script
	if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
		return "", fmt.Errorf("falha ao criar wrapper: %w", err)
	}
	
	// No Windows, verificar se o diretório está no PATH e adicionar instruções
	if util.IsWindows() {
		pathEnv := os.Getenv("PATH")
		if !strings.Contains(pathEnv, binDir) {
			logger.Warning("O diretório %s não está no seu PATH", binDir)
			logger.Info("Adicione-o manualmente ou use o caminho completo: %s", wrapperPath)
		}
	}
	
	return wrapperPath, nil
}

// installWhisperCPP instala a versão C++ do Whisper
func installWhisperCPP(logger *util.Logger) (string, error) {
	logger.Info("Instalando Whisper C++...")
	
	// Verificar dependências de compilação
	if err := checkCppBuildDependencies(); err != nil {
		return "", fmt.Errorf("dependências de compilação não encontradas: %w", err)
	}
	
	// Determinar diretório para instalação
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar o diretório home: %w", err)
	}
	
	whisperDir := filepath.Join(homeDir, ".local", "share", "whisper-cpp")
	if util.IsWindows() {
		whisperDir = filepath.Join(homeDir, "AppData", "Local", "CompressVideo", "whisper-cpp")
	}
	
	// Criar diretório
	if err := os.MkdirAll(whisperDir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório: %w", err)
	}
	
	// Clonar repositório
	logger.Info("Clonando repositório whisper.cpp...")
	cmd := exec.Command("git", "clone", "https://github.com/ggerganov/whisper.cpp.git", whisperDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Se já existe, tentar atualizar
		if strings.Contains(string(output), "already exists") {
			logger.Info("Repositório já existe, atualizando...")
			cmd = exec.Command("git", "-C", whisperDir, "pull")
			if _, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("falha ao atualizar repositório: %w", err)
			}
		} else {
			return "", fmt.Errorf("falha ao clonar repositório: %w\nOutput: %s", err, output)
		}
	}
	
	// Compilar
	logger.Info("Compilando whisper.cpp...")
	cmd = exec.Command("make", "-C", whisperDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("falha ao compilar: %w\nOutput: %s", err, output)
	}
	
	// Criar link simbólico ou copiar para PATH
	binDir := filepath.Join(homeDir, ".local", "bin")
	if util.IsWindows() {
		binDir = filepath.Join(homeDir, "AppData", "Local", "CompressVideo", "bin")
	}
	
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório bin: %w", err)
	}
	
	whisperBin := filepath.Join(whisperDir, "main")
	if util.IsWindows() {
		whisperBin += ".exe"
	}
	
	destPath := filepath.Join(binDir, "whisper")
	if util.IsWindows() {
		destPath += ".exe"
		// No Windows, copiar o executável
		if err := copyFile(whisperBin, destPath); err != nil {
			return "", fmt.Errorf("falha ao copiar executável: %w", err)
		}
	} else {
		// No Linux/Mac, criar link simbólico
		os.Remove(destPath) // Remover link existente se houver
		if err := os.Symlink(whisperBin, destPath); err != nil {
			return "", fmt.Errorf("falha ao criar link simbólico: %w", err)
		}
	}
	
	return destPath, nil
}

// checkCppBuildDependencies verifica se as dependências para compilar whisper.cpp estão disponíveis
func checkCppBuildDependencies() error {
	deps := []string{"git", "make", "g++"}
	if runtime.GOOS == "darwin" {
		deps = []string{"git", "make", "clang++"}
	} else if runtime.GOOS == "windows" {
		// No Windows, precisamos do MSYS2/MinGW ou equivalente
		deps = []string{"git", "make", "gcc"}
	}
	
	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("dependência não encontrada: %s", dep)
		}
	}
	
	return nil
}

// copyFile copia um arquivo de src para dst
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0755)
} 