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
	
	// Escolher o método de instalação com base nas capacidades do sistema
	if canUsePython() {
		// Instalar a versão Python do Whisper
		whisperPath, err := installWhisperPython(wi.Logger)
		if err != nil {
			wi.Logger.Error("Falha ao instalar Whisper (Python): %v", err)
			wi.Logger.Info("Tentando instalar versão C++...")
			
			// Falhou com Python, tenta C++
			whisperPath, err = installWhisperCPP(wi.Logger)
			if err != nil {
				return fmt.Errorf("falha ao instalar Whisper: %w", err)
			}
		}
		
		wi.Logger.Success("Whisper instalado com sucesso em: %s", whisperPath)
		return nil
	}
	
	// Não tem Python, tenta C++
	whisperPath, err := installWhisperCPP(wi.Logger)
	if err != nil {
		return fmt.Errorf("falha ao instalar Whisper: %w", err)
	}
	
	wi.Logger.Success("Whisper instalado com sucesso em: %s", whisperPath)
	return nil
}

// canUsePython verifica se o Python está disponível no sistema
func canUsePython() bool {
	_, err := exec.LookPath("python3")
	if err == nil {
		return true
	}
	
	_, err = exec.LookPath("python")
	return err == nil
}

// installWhisperPython instala o Whisper usando pip
func installWhisperPython(logger *util.Logger) (string, error) {
	logger.Info("Instalando Whisper usando Python...")
	
	// Determinar qual Python usar
	pythonCmd := "python3"
	_, err := exec.LookPath(pythonCmd)
	if err != nil {
		pythonCmd = "python"
		_, err = exec.LookPath(pythonCmd)
		if err != nil {
			return "", fmt.Errorf("Python não encontrado no sistema")
		}
	}
	
	// Instalar Whisper via pip
	logger.Info("Instalando pacote openai-whisper...")
	cmd := exec.Command(pythonCmd, "-m", "pip", "install", "openai-whisper")
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
		wrapperPath += ".cmd"
	}
	
	// Conteúdo do script
	var content string
	if util.IsWindows() {
		content = fmt.Sprintf("@echo off\n%s -m whisper %%*", pythonCmd)
	} else {
		content = fmt.Sprintf("#!/bin/sh\n%s -m whisper \"$@\"", pythonCmd)
	}
	
	// Escrever o script
	if err := os.WriteFile(wrapperPath, []byte(content), 0755); err != nil {
		return "", fmt.Errorf("falha ao criar wrapper: %w", err)
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
	}
	
	for _, dep := range deps {
		if _, err := exec.LookPath(dep); err != nil {
			return fmt.Errorf("dependência não encontrada: %s", dep)
		}
	}
	
	return nil
}

// copyFile copia um arquivo de origem para destino
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	
	err = os.WriteFile(dst, input, 0755)
	if err != nil {
		return err
	}
	
	return nil
} 