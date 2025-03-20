package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// FFmpegDownloadBase é a URL base para FFmpeg estático
	FFmpegDownloadBase = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest"
	
	// Versões estáticas do FFmpeg por plataforma
	ffmpegWin64  = "ffmpeg-master-latest-win64-gpl.zip"
	ffmpegLinux64 = "ffmpeg-master-latest-linux64-gpl.tar.xz"
	ffmpegMacOS  = "ffmpeg-master-latest-macos64-gpl.tar.xz"
)

// FFmpegInfo contém informações sobre o FFmpeg instalado
type FFmpegInfo struct {
	Available    bool   // Se o FFmpeg está disponível
	Path         string // Caminho para o executável do FFmpeg
	Version      string // Versão do FFmpeg
	IsDownloaded bool   // Se esta é uma versão baixada por nós
}

// FindFFmpeg verifica se o FFmpeg está instalado no sistema e retorna informações sobre ele
func FindFFmpeg() (*FFmpegInfo, error) {
	info := &FFmpegInfo{
		Available: false,
	}
	
	// Primeiro, verificar se o FFmpeg já está disponível no PATH
	path, err := exec.LookPath("ffmpeg")
	if err == nil {
		// FFmpeg encontrado no PATH
		info.Available = true
		info.Path = path
		info.Version = getFFmpegVersion(path)
		return info, nil
	}
	
	// Segundo, verificar se já baixamos o FFmpeg anteriormente
	userDir, err := os.UserHomeDir()
	if err != nil {
		return info, fmt.Errorf("não foi possível determinar o diretório do usuário: %w", err)
	}
	
	// Verificar no diretório .compressvideo
	appDir := filepath.Join(userDir, ".compressvideo")
	ffmpegPath := filepath.Join(appDir, "bin", "ffmpeg"+GetExecutableExtension())
	
	if _, err := os.Stat(ffmpegPath); err == nil {
		// FFmpeg encontrado no diretório da aplicação
		info.Available = true
		info.Path = ffmpegPath
		info.Version = getFFmpegVersion(ffmpegPath)
		info.IsDownloaded = true
		return info, nil
	}
	
	// FFmpeg não encontrado
	return info, nil
}

// EnsureFFmpeg garante que o FFmpeg está disponível, baixando-o se necessário
func EnsureFFmpeg(logger *Logger) (*FFmpegInfo, error) {
	info, err := FindFFmpeg()
	if err != nil {
		return nil, err
	}
	
	if info.Available {
		logger.Info("FFmpeg encontrado: %s", info.Path)
		return info, nil
	}
	
	// FFmpeg não está disponível, precisamos baixá-lo
	logger.Warning("FFmpeg não encontrado. Tentando baixar uma versão adequada...")
	
	downloadedPath, err := DownloadFFmpeg(logger)
	if err != nil {
		return nil, fmt.Errorf("falha ao baixar FFmpeg: %w", err)
	}
	
	// Atualizar as informações
	info.Available = true
	info.Path = downloadedPath
	info.Version = getFFmpegVersion(downloadedPath)
	info.IsDownloaded = true
	
	logger.Success("FFmpeg baixado com sucesso: %s", info.Path)
	return info, nil
}

// DownloadFFmpeg baixa e extrai uma versão apropriada do FFmpeg
func DownloadFFmpeg(logger *Logger) (string, error) {
	// Determinar qual arquivo baixar com base no sistema operacional
	var downloadURL, filename string
	
	switch runtime.GOOS {
	case "windows":
		filename = ffmpegWin64
	case "linux":
		filename = ffmpegLinux64
	case "darwin": // macOS
		filename = ffmpegMacOS
	default:
		return "", fmt.Errorf("sistema operacional não suportado para download automático: %s", runtime.GOOS)
	}
	
	downloadURL = FFmpegDownloadBase + "/" + filename
	
	// Criar diretório para armazenar o FFmpeg
	userDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar o diretório do usuário: %w", err)
	}
	
	appDir := filepath.Join(userDir, ".compressvideo")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", fmt.Errorf("não foi possível criar diretório para o FFmpeg: %w", err)
	}
	
	downloadPath := filepath.Join(appDir, filename)
	binDir := filepath.Join(appDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("não foi possível criar diretório bin: %w", err)
	}
	
	// Baixar o arquivo
	logger.Info("Baixando FFmpeg de %s...", downloadURL)
	if err := downloadFile(downloadURL, downloadPath, logger); err != nil {
		return "", fmt.Errorf("falha ao baixar FFmpeg: %w", err)
	}
	
	// Extrair o arquivo
	logger.Info("Extraindo FFmpeg...")
	ffmpegPath, err := extractFFmpeg(downloadPath, binDir, logger)
	if err != nil {
		return "", fmt.Errorf("falha ao extrair FFmpeg: %w", err)
	}
	
	// Remover o arquivo baixado para economizar espaço
	os.Remove(downloadPath)
	
	return ffmpegPath, nil
}

// downloadFile baixa um arquivo da URL especificada
func downloadFile(url, destPath string, logger *Logger) error {
	// Criar o arquivo de destino
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	
	// Fazer a requisição HTTP
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Verificar o código de status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status de resposta inválido: %s", resp.Status)
	}
	
	// Criar barra de progresso se o tamanho for conhecido
	var reader io.Reader = resp.Body
	if resp.ContentLength > 0 {
		progressBar := NewProgressTracker(resp.ContentLength, "Baixando FFmpeg", logger)
		reader = &progressReader{reader: resp.Body, tracker: progressBar}
		defer progressBar.Finish()
	}
	
	// Copiar o conteúdo para o arquivo
	_, err = io.Copy(out, reader)
	return err
}

// extractFFmpeg extrai o FFmpeg do arquivo baixado
func extractFFmpeg(archivePath, destDir string, logger *Logger) (string, error) {
	// Determinar o tipo de arquivo
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir, logger)
	} else if strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz") {
		return extractTarGz(archivePath, destDir, logger)
	} else if strings.HasSuffix(archivePath, ".tar.xz") {
		// Para arquivos .tar.xz, precisamos usar comandos externos como tar com suporte a xz
		return extractWithExternalCommand(archivePath, destDir, logger)
	}
	
	return "", errors.New("formato de arquivo não suportado")
}

// extractZip extrai um arquivo zip
func extractZip(zipPath, destDir string, logger *Logger) (string, error) {
	// Abrir o arquivo zip
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	
	// Procurar o executável do ffmpeg
	var ffmpegPath string
	
	for _, file := range reader.File {
		// Verificar se é o executável do ffmpeg
		if filepath.Base(file.Name) == "ffmpeg"+GetExecutableExtension() {
			// Extrair o arquivo
			src, err := file.Open()
			if err != nil {
				return "", err
			}
			defer src.Close()
			
			// Caminho de destino para o ffmpeg
			ffmpegPath = filepath.Join(destDir, "ffmpeg"+GetExecutableExtension())
			dst, err := os.OpenFile(ffmpegPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return "", err
			}
			defer dst.Close()
			
			// Copiar o conteúdo
			_, err = io.Copy(dst, src)
			if err != nil {
				return "", err
			}
			
			break
		}
	}
	
	if ffmpegPath == "" {
		return "", errors.New("executável do ffmpeg não encontrado no arquivo zip")
	}
	
	// Tornar o arquivo executável
	if err := os.Chmod(ffmpegPath, 0755); err != nil {
		return "", err
	}
	
	return ffmpegPath, nil
}

// extractTarGz extrai um arquivo tar.gz
func extractTarGz(tarPath, destDir string, logger *Logger) (string, error) {
	// Abrir o arquivo
	file, err := os.Open(tarPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	// Criar um leitor gzip
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()
	
	// Criar um leitor tar
	tr := tar.NewReader(gzr)
	
	// Procurar o executável do ffmpeg
	var ffmpegPath string
	
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		
		// Verificar se é o executável do ffmpeg
		if filepath.Base(header.Name) == "ffmpeg"+GetExecutableExtension() {
			// Extrair o arquivo
			ffmpegPath = filepath.Join(destDir, "ffmpeg"+GetExecutableExtension())
			dst, err := os.OpenFile(ffmpegPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}
			defer dst.Close()
			
			// Copiar o conteúdo
			_, err = io.Copy(dst, tr)
			if err != nil {
				return "", err
			}
			
			break
		}
	}
	
	if ffmpegPath == "" {
		return "", errors.New("executável do ffmpeg não encontrado no arquivo tar.gz")
	}
	
	// Tornar o arquivo executável
	if err := os.Chmod(ffmpegPath, 0755); err != nil {
		return "", err
	}
	
	return ffmpegPath, nil
}

// extractWithExternalCommand extrai arquivos usando comandos externos (para tar.xz)
func extractWithExternalCommand(archivePath, destDir string, logger *Logger) (string, error) {
	// Verificar se temos o tar instalado
	tarPath, err := exec.LookPath("tar")
	if err != nil {
		return "", errors.New("o comando 'tar' é necessário para extrair arquivos .tar.xz, mas não foi encontrado")
	}
	
	// Criar diretório temporário para extração completa
	tempDir, err := os.MkdirTemp("", "ffmpeg-extract-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)
	
	// Extrair o arquivo com tar
	cmd := exec.Command(tarPath, "-xf", archivePath, "-C", tempDir)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("falha ao extrair com tar: %w", err)
	}
	
	// Procurar o executável do ffmpeg recursivamente
	var ffmpegPath string
	
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && filepath.Base(path) == "ffmpeg"+GetExecutableExtension() {
			// Encontrou o ffmpeg
			ffmpegDestPath := filepath.Join(destDir, "ffmpeg"+GetExecutableExtension())
			
			// Copiar para o diretório de destino
			src, err := os.Open(path)
			if err != nil {
				return err
			}
			defer src.Close()
			
			dst, err := os.OpenFile(ffmpegDestPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
			if err != nil {
				return err
			}
			defer dst.Close()
			
			if _, err := io.Copy(dst, src); err != nil {
				return err
			}
			
			// Tornar o arquivo executável
			if err := os.Chmod(ffmpegDestPath, 0755); err != nil {
				return err
			}
			
			ffmpegPath = ffmpegDestPath
		}
		
		return nil
	})
	
	if err != nil {
		return "", err
	}
	
	if ffmpegPath == "" {
		return "", errors.New("executável do ffmpeg não encontrado após extração")
	}
	
	return ffmpegPath, nil
}

// getFFmpegVersion retorna a versão do FFmpeg
func getFFmpegVersion(ffmpegPath string) string {
	cmd := exec.Command(ffmpegPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "desconhecida"
	}
	
	// Extrai a primeira linha que geralmente contém a versão
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	
	return "desconhecida"
}

// progressReader é um leitor que rastreia o progresso
type progressReader struct {
	reader  io.Reader
	tracker *ProgressTracker
	total   int64
}

// Read implementa a interface io.Reader
func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.total += int64(n)
		pr.tracker.Update(pr.total)
	}
	return n, err
} 