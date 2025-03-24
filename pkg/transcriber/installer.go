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
	// Lista de possíveis comandos Python
	pythonCommands := []string{"python3", "python"}
	
	// No Windows, verificar caminhos comuns
	if util.IsWindows() {
		// Adicionar caminhos comuns de instalação do Python no Windows
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" {
			pythonDirs, _ := filepath.Glob(filepath.Join(userProfile, "AppData", "Local", "Programs", "Python", "Python*"))
			for _, dir := range pythonDirs {
				pythonCommands = append(pythonCommands, filepath.Join(dir, "python.exe"))
			}
		}
		
		// Verificar também em C:\Python*
		pythonDirs, _ := filepath.Glob("C:\\Python*")
		for _, dir := range pythonDirs {
			pythonCommands = append(pythonCommands, filepath.Join(dir, "python.exe"))
		}
		
		// Verificar também o ProgramFiles
		progFiles := os.Getenv("ProgramFiles")
		if progFiles != "" {
			pythonDirs, _ := filepath.Glob(filepath.Join(progFiles, "Python*"))
			for _, dir := range pythonDirs {
				pythonCommands = append(pythonCommands, filepath.Join(dir, "python.exe"))
			}
		}
	}
	
	// Testar cada comando
	for _, cmd := range pythonCommands {
		if path, err := exec.LookPath(cmd); err == nil {
			// Verificar se é um Python real e não um alias do Windows Store
			if util.IsWindows() {
				// Testar executando um comando simples
				versionCmd := exec.Command(path, "--version")
				if output, err := versionCmd.CombinedOutput(); err == nil && strings.Contains(string(output), "Python") {
					return path, nil
				}
			} else {
				return path, nil
			}
		}
	}
	
	return "", fmt.Errorf("não foi possível encontrar um comando Python válido")
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
	
	binDir := filepath.Join(homeDir, ".local", "bin")
	if util.IsWindows() {
		binDir = filepath.Join(homeDir, "AppData", "Local", "CompressVideo", "bin")
	}
	
	// Criar diretório se não existir
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("falha ao criar diretório: %w", err)
	}
	
	// Caminho para o whisper-ctranslate2
	whisperCtranslate2Path := ""
	scriptsDirPath := filepath.Join(filepath.Dir(pythonCmd), "Scripts")
	
	if util.IsWindows() {
		// Procurar pelo executável whisper-ctranslate2 diretamente
		whisperCtranslateDirect := filepath.Join(scriptsDirPath, "whisper-ctranslate2.exe")
		if _, err := os.Stat(whisperCtranslateDirect); err == nil {
			whisperCtranslate2Path = whisperCtranslateDirect
		}
	}
	
	// Caminho para o script wrapper
	wrapperPath := filepath.Join(binDir, "whisper")
	if util.IsWindows() {
		wrapperPath += ".bat"
	}
	
	// Conteúdo do script
	var content string
	if util.IsWindows() {
		if whisperCtranslate2Path != "" {
			content = fmt.Sprintf("@echo off\n\"%s\" %%*", whisperCtranslate2Path)
		} else {
			content = fmt.Sprintf("@echo off\n\"%s\" -m whisper_ctranslate2 %%*", pythonCmd)
		}
	} else {
		content = fmt.Sprintf("#!/bin/sh\n%s -m whisper_ctranslate2 \"$@\"", pythonCmd)
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