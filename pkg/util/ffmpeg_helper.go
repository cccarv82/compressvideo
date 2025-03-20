package util

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// FFmpegDownloadBase é a URL base para FFmpeg estático
	FFmpegDownloadBase = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest"
	
	// Versões estáticas do FFmpeg por plataforma
	ffmpegWin64  = "ffmpeg-master-latest-win64-gpl.zip"
	ffmpegLinux64 = "ffmpeg-master-latest-linux64-gpl.tar.xz"
	ffmpegMacOS  = "ffmpeg-master-latest-macos64-gpl.tar.xz"
)

// FFmpegInfo contém informações sobre a instalação do FFmpeg
type FFmpegInfo struct {
	Available    bool   // Se o FFmpeg está disponível
	Path         string // Caminho para o executável do FFmpeg
	FFprobePath  string // Caminho para o executável do FFprobe
	Version      string // Versão do FFmpeg
	IsDownloaded bool   // Se esta é uma versão baixada por nós
}

// URLs de download para diferentes sistemas operacionais
var ffmpegMirrors = map[OSType][]string{
	Windows: {
		"https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip",
		"https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip",
		"https://github.com/GyanD/codexffmpeg/releases/download/5.1.2/ffmpeg-5.1.2-essentials_build.zip",
	},
	Linux: {
		"https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz",
		"https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz",
	},
	MacOS: {
		"https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip",
		"https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip",
	},
}

// Diretório onde o FFmpeg será armazenado
func getFFmpegDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	
	return filepath.Join(homeDir, ".compressvideo")
}

// FindFFmpeg procura pelo FFmpeg no sistema
func FindFFmpeg() (*FFmpegInfo, error) {
	// Primeiro, verifica se temos uma versão baixada
	downloadedPath := filepath.Join(getFFmpegDir(), "bin", "ffmpeg"+GetExecutableExtension())
	downloadedProbePath := filepath.Join(getFFmpegDir(), "bin", "ffprobe"+GetExecutableExtension())
	
	if fileExists(downloadedPath) && fileExists(downloadedProbePath) {
		// Testar se a versão baixada funciona
		if err := testFFmpegInstallation(downloadedPath, downloadedProbePath); err == nil {
			version := getFFmpegVersion(downloadedPath)
			return &FFmpegInfo{
				Available:    true,
				Path:         downloadedPath,
				FFprobePath:  downloadedProbePath,
				Version:      version,
				IsDownloaded: true,
			}, nil
		}
	}
	
	// Se não encontrou a versão baixada, procura no PATH
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err == nil {
		// Encontrou o FFmpeg, agora procura o FFprobe
		ffprobePath, probErr := exec.LookPath("ffprobe")
		if probErr == nil {
			// Testar se funciona
			if err := testFFmpegInstallation(ffmpegPath, ffprobePath); err == nil {
				version := getFFmpegVersion(ffmpegPath)
				return &FFmpegInfo{
					Available:    true,
					Path:         ffmpegPath,
					FFprobePath:  ffprobePath,
					Version:      version,
					IsDownloaded: false,
				}, nil
			}
		}
	}
	
	// Não encontrou FFmpeg ou não passou nos testes
	return &FFmpegInfo{
		Available: false,
	}, nil
}

// EnsureFFmpeg garante que o FFmpeg está disponível, baixando se necessário
func EnsureFFmpeg(logger *Logger) (*FFmpegInfo, error) {
	// Tenta encontrar o FFmpeg
	info, err := FindFFmpeg()
	if err != nil {
		return nil, err
	}
	
	// Se já está disponível, retorna
	if info.Available {
		logger.Info("FFmpeg encontrado: %s", info.Path)
		return info, nil
	}
	
	// FFmpeg não encontrado, baixar
	logger.Info("FFmpeg não encontrado. Baixando automaticamente...")
	ffmpegPath, ffprobePath, err := DownloadFFmpeg(logger)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar FFmpeg: %v", err)
	}
	
	// Verificar se o FFmpeg foi baixado corretamente
	if err := testFFmpegInstallation(ffmpegPath, ffprobePath); err != nil {
		return nil, fmt.Errorf("erro ao verificar FFmpeg baixado: %v", err)
	}
	
	version := getFFmpegVersion(ffmpegPath)
	logger.Success("FFmpeg baixado com sucesso: %s (versão %s)", ffmpegPath, version)
	
	return &FFmpegInfo{
		Available:    true,
		Path:         ffmpegPath,
		FFprobePath:  ffprobePath,
		Version:      version,
		IsDownloaded: true,
	}, nil
}

// RepairFFmpeg força o download e reinstalação do FFmpeg
func RepairFFmpeg(logger *Logger) (*FFmpegInfo, error) {
	// Remover a instalação existente
	ffmpegDir := getFFmpegDir()
	if err := os.RemoveAll(ffmpegDir); err != nil {
		logger.Warning("Falha ao remover diretório do FFmpeg: %v", err)
		// Continua mesmo se falhar a remoção
	}

	// Baixar novamente
	logger.Info("Reinstalando FFmpeg...")
	ffmpegPath, ffprobePath, err := DownloadFFmpeg(logger)
	if err != nil {
		return nil, fmt.Errorf("falha ao reinstalar FFmpeg: %v", err)
	}

	// Verificar instalação
	if err := testFFmpegInstallation(ffmpegPath, ffprobePath); err != nil {
		return nil, fmt.Errorf("FFmpeg instalado, mas com problemas: %v", err)
	}

	version := getFFmpegVersion(ffmpegPath)
	logger.Success("FFmpeg reinstalado com sucesso: %s (versão %s)", ffmpegPath, version)

	return &FFmpegInfo{
		Available:    true,
		Path:         ffmpegPath,
		FFprobePath:  ffprobePath,
		Version:      version,
		IsDownloaded: true,
	}, nil
}

// DownloadFFmpeg baixa e instala o FFmpeg
func DownloadFFmpeg(logger *Logger) (string, string, error) {
	osType := GetCurrentOS()
	if osType == Unknown {
		return "", "", fmt.Errorf("sistema operacional não suportado: %s", runtime.GOOS)
	}
	
	// Criar diretório para armazenar o FFmpeg
	ffmpegDir := getFFmpegDir()
	binDir := filepath.Join(ffmpegDir, "bin")
	tempDir := filepath.Join(ffmpegDir, "temp")
	
	// Limpar diretório temporário, mas criar novos
	os.RemoveAll(tempDir)
	
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", "", fmt.Errorf("falha ao criar diretório para FFmpeg: %v", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", "", fmt.Errorf("falha ao criar diretório temporário: %v", err)
	}
	
	// Definir caminhos para os executáveis
	ffmpegExe := filepath.Join(binDir, "ffmpeg"+GetExecutableExtension())
	ffprobeExe := filepath.Join(binDir, "ffprobe"+GetExecutableExtension())
	
	// Se já existem e funcionam, retorna
	if fileExists(ffmpegExe) && fileExists(ffprobeExe) {
		if err := testFFmpegInstallation(ffmpegExe, ffprobeExe); err == nil {
			logger.Info("FFmpeg já instalado em: %s", ffmpegExe)
			return ffmpegExe, ffprobeExe, nil
		} else {
			logger.Warning("FFmpeg existente com problemas: %v", err)
			// Vai tentar baixar novamente
		}
	}
	
	// Baixar de múltiplas fontes
	var lastError error
	var archivePath string
	
	for _, url := range ffmpegMirrors[osType] {
		logger.Info("Tentando baixar FFmpeg de: %s", url)
		
		archivePath = filepath.Join(tempDir, "ffmpeg-temp"+getArchiveExtension(url))
		
		// Tenta baixar com timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		err := downloadFileWithProgress(ctx, url, archivePath, logger)
		cancel()
		
		if err != nil {
			logger.Warning("Falha ao baixar de %s: %v", url, err)
			lastError = err
			continue
		}
		
		// Verificar se o arquivo foi baixado corretamente
		if !fileExists(archivePath) || getFileSize(archivePath) < 1000000 {
			logger.Warning("Arquivo baixado inválido ou muito pequeno")
			lastError = fmt.Errorf("arquivo baixado inválido")
			continue
		}
		
		// Sucesso no download, parar de tentar
		logger.Info("Download concluído com sucesso")
		break
	}
	
	// Se não conseguiu baixar de nenhuma fonte
	if archivePath == "" || !fileExists(archivePath) {
		return "", "", fmt.Errorf("falha ao baixar FFmpeg de todas as fontes: %v", lastError)
	}
	
	// Extrair o arquivo baixado
	logger.Info("Extraindo FFmpeg...")
	
	var err error
	switch osType {
	case Windows:
		err = extractWindowsFFmpeg(archivePath, binDir, logger)
	case MacOS:
		err = extractMacOSFFmpeg(archivePath, binDir, logger)
	case Linux:
		err = extractLinuxFFmpeg(archivePath, binDir, logger)
	}
	
	if err != nil {
		return "", "", fmt.Errorf("falha ao extrair FFmpeg: %v", err)
	}
	
	// Verificar se os executáveis existem
	if !fileExists(ffmpegExe) {
		return "", "", fmt.Errorf("ffmpeg não encontrado após extração")
	}
	
	if !fileExists(ffprobeExe) {
		// No macOS podemos precisar baixar o FFprobe separadamente
		if osType == MacOS {
			logger.Info("FFprobe não encontrado, tentando baixar separadamente...")
			err = downloadMacOSFFprobe(binDir, logger)
			if err != nil {
				return "", "", fmt.Errorf("falha ao baixar FFprobe para macOS: %v", err)
			}
		} else {
			return "", "", fmt.Errorf("ffprobe não encontrado após extração")
		}
	}
	
	// Definir permissões de execução
	if osType != Windows {
		os.Chmod(ffmpegExe, 0755)
		os.Chmod(ffprobeExe, 0755)
	}
	
	// Testar instalação
	if err := testFFmpegInstallation(ffmpegExe, ffprobeExe); err != nil {
		return "", "", fmt.Errorf("FFmpeg instalado, mas com problemas: %v", err)
	}
	
	// Limpar arquivos temporários
	os.RemoveAll(tempDir)
	
	return ffmpegExe, ffprobeExe, nil
}

// Função específica para extrair FFmpeg no Windows
func extractWindowsFFmpeg(zipPath, destDir string, logger *Logger) error {
	// Ler o arquivo zip
	logger.Info("Extraindo arquivo zip em: %s", destDir)
	
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo zip: %v", err)
	}
	defer archive.Close()
	
	// Procurar os executáveis ffmpeg.exe e ffprobe.exe em qualquer subdiretório
	var ffmpegFound, ffprobeFound bool
	
	for _, file := range archive.File {
		fileName := filepath.Base(file.Name)
		
		// Verificar se é ffmpeg.exe ou ffprobe.exe
		if strings.EqualFold(fileName, "ffmpeg.exe") || strings.EqualFold(fileName, "ffprobe.exe") {
			// Abrir o arquivo do zip
			srcFile, err := file.Open()
			if err != nil {
				logger.Warning("Falha ao abrir %s do zip: %v", fileName, err)
				continue
			}
			
			// Criar o arquivo de destino
			destPath := filepath.Join(destDir, fileName)
			destFile, err := os.Create(destPath)
			if err != nil {
				srcFile.Close()
				logger.Warning("Falha ao criar %s: %v", destPath, err)
				continue
			}
			
			// Copiar o conteúdo
			_, err = io.Copy(destFile, srcFile)
			srcFile.Close()
			destFile.Close()
			
			if err != nil {
				logger.Warning("Falha ao copiar %s: %v", fileName, err)
				continue
			}
			
			if strings.EqualFold(fileName, "ffmpeg.exe") {
				ffmpegFound = true
				logger.Info("FFmpeg extraído para: %s", destPath)
			} else {
				ffprobeFound = true
				logger.Info("FFprobe extraído para: %s", destPath)
			}
			
			// Se encontrou ambos, pode parar
			if ffmpegFound && ffprobeFound {
				break
			}
		}
	}
	
	if !ffmpegFound || !ffprobeFound {
		// Tentar encontrar em subdiretórios específicos comuns em diferentes distribuições
		searchDirs := []string{"bin", "ffmpeg-*-win64-static/bin", "ffmpeg-*-essentials_build/bin"}
		
		for _, searchPath := range searchDirs {
			logger.Debug("Procurando executáveis em: %s", searchPath)
			
			for _, file := range archive.File {
				if !strings.Contains(strings.ToLower(file.Name), strings.ToLower(searchPath)) {
					continue
				}
				
				fileName := filepath.Base(file.Name)
				if strings.EqualFold(fileName, "ffmpeg.exe") || strings.EqualFold(fileName, "ffprobe.exe") {
					// Extrair o arquivo
					srcFile, err := file.Open()
					if err != nil {
						logger.Warning("Falha ao abrir %s do zip: %v", fileName, err)
						continue
					}
					
					destPath := filepath.Join(destDir, fileName)
					destFile, err := os.Create(destPath)
					if err != nil {
						srcFile.Close()
						logger.Warning("Falha ao criar %s: %v", destPath, err)
						continue
					}
					
					_, err = io.Copy(destFile, srcFile)
					srcFile.Close()
					destFile.Close()
					
					if err != nil {
						logger.Warning("Falha ao copiar %s: %v", fileName, err)
						continue
					}
					
					if strings.EqualFold(fileName, "ffmpeg.exe") {
						ffmpegFound = true
						logger.Info("FFmpeg extraído para: %s", destPath)
					} else {
						ffprobeFound = true
						logger.Info("FFprobe extraído para: %s", destPath)
					}
					
					if ffmpegFound && ffprobeFound {
						break
					}
				}
			}
			
			if ffmpegFound && ffprobeFound {
				break
			}
		}
	}
	
	if !ffmpegFound {
		return fmt.Errorf("ffmpeg.exe não encontrado no arquivo zip")
	}
	
	if !ffprobeFound {
		return fmt.Errorf("ffprobe.exe não encontrado no arquivo zip")
	}
	
	return nil
}

// Função específica para extrair FFmpeg no Linux
func extractLinuxFFmpeg(archivePath, destDir string, logger *Logger) error {
	// No Linux, normalmente é um arquivo tar.xz ou tar.gz
	// Usar o comando tar externo é mais fácil e confiável
	
	logger.Info("Extraindo arquivo para: %s", destDir)
	
	// Criar diretório temporário para extração
	tempExtractDir := filepath.Join(filepath.Dir(archivePath), "extracted")
	os.MkdirAll(tempExtractDir, 0755)
	
	// Comando para extrair
	var cmd *exec.Cmd
	if strings.HasSuffix(archivePath, ".tar.xz") {
		cmd = exec.Command("tar", "-xf", archivePath, "-C", tempExtractDir)
	} else if strings.HasSuffix(archivePath, ".tar.gz") {
		cmd = exec.Command("tar", "-xzf", archivePath, "-C", tempExtractDir)
	} else {
		return fmt.Errorf("formato de arquivo não suportado: %s", archivePath)
	}
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("falha ao extrair com tar: %v (output: %s)", err, string(output))
	}
	
	// Procurar os executáveis ffmpeg e ffprobe
	err = filepath.Walk(tempExtractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			fileName := filepath.Base(path)
			
			if fileName == "ffmpeg" {
				destPath := filepath.Join(destDir, "ffmpeg")
				copyFile(path, destPath)
				os.Chmod(destPath, 0755)
				logger.Info("FFmpeg copiado para: %s", destPath)
			} else if fileName == "ffprobe" {
				destPath := filepath.Join(destDir, "ffprobe")
				copyFile(path, destPath)
				os.Chmod(destPath, 0755)
				logger.Info("FFprobe copiado para: %s", destPath)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("erro ao procurar executáveis: %v", err)
	}
	
	// Verificar se os arquivos foram copiados
	if !fileExists(filepath.Join(destDir, "ffmpeg")) {
		return fmt.Errorf("ffmpeg não encontrado após extração")
	}
	
	if !fileExists(filepath.Join(destDir, "ffprobe")) {
		return fmt.Errorf("ffprobe não encontrado após extração")
	}
	
	return nil
}

// Função específica para extrair FFmpeg no macOS
func extractMacOSFFmpeg(archivePath, destDir string, logger *Logger) error {
	// No macOS, geralmente é um zip com um único executável
	
	logger.Info("Extraindo FFmpeg para macOS em: %s", destDir)
	
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo zip: %v", err)
	}
	defer archive.Close()
	
	// O arquivo zip do evermeet.cx geralmente tem apenas um arquivo: ffmpeg
	for _, file := range archive.File {
		// Extrair todos os arquivos (normalmente é apenas um)
		srcFile, err := file.Open()
		if err != nil {
			logger.Warning("Falha ao abrir arquivo do zip: %v", err)
			continue
		}
		
		destPath := filepath.Join(destDir, file.Name)
		destFile, err := os.Create(destPath)
		if err != nil {
			srcFile.Close()
			logger.Warning("Falha ao criar arquivo: %v", err)
			continue
		}
		
		_, err = io.Copy(destFile, srcFile)
		srcFile.Close()
		destFile.Close()
		
		if err != nil {
			logger.Warning("Falha ao copiar arquivo: %v", err)
			continue
		}
		
		// Se o arquivo é ffmpeg, renomear para ffmpeg sem versão
		if strings.HasPrefix(file.Name, "ffmpeg") {
			os.Rename(destPath, filepath.Join(destDir, "ffmpeg"))
			os.Chmod(filepath.Join(destDir, "ffmpeg"), 0755)
			logger.Info("FFmpeg extraído para: %s", filepath.Join(destDir, "ffmpeg"))
		}
	}
	
	return nil
}

// Função específica para baixar FFprobe para macOS
func downloadMacOSFFprobe(destDir string, logger *Logger) error {
	// No macOS, FFprobe é frequentemente distribuído separadamente
	
	logger.Info("Baixando FFprobe para macOS...")
	
	tempDir := filepath.Join(destDir, "temp")
	os.MkdirAll(tempDir, 0755)
	
	archivePath := filepath.Join(tempDir, "ffprobe.zip")
	
	// Tenta baixar de fontes específicas para macOS
	var lastError error
	for _, url := range []string{
		"https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip",
		"https://evermeet.cx/ffmpeg/ffprobe-5.1.zip",
	} {
		logger.Info("Tentando baixar FFprobe de: %s", url)
		
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		err := downloadFileWithProgress(ctx, url, archivePath, logger)
		cancel()
		
		if err != nil {
			logger.Warning("Falha ao baixar de %s: %v", url, err)
			lastError = err
			continue
		}
		
		// Extrair o arquivo zip
		archive, err := zip.OpenReader(archivePath)
		if err != nil {
			logger.Warning("Falha ao abrir arquivo zip: %v", err)
			lastError = err
			continue
		}
		
		for _, file := range archive.File {
			// Extrair todos os arquivos
			srcFile, err := file.Open()
			if err != nil {
				logger.Warning("Falha ao abrir arquivo do zip: %v", err)
				continue
			}
			
			destPath := filepath.Join(destDir, file.Name)
			destFile, err := os.Create(destPath)
			if err != nil {
				srcFile.Close()
				logger.Warning("Falha ao criar arquivo: %v", err)
				continue
			}
			
			_, err = io.Copy(destFile, srcFile)
			srcFile.Close()
			destFile.Close()
			
			if err != nil {
				logger.Warning("Falha ao copiar arquivo: %v", err)
				continue
			}
			
			// Se o arquivo é ffprobe, renomear para ffprobe sem versão
			if strings.HasPrefix(file.Name, "ffprobe") {
				os.Rename(destPath, filepath.Join(destDir, "ffprobe"))
				os.Chmod(filepath.Join(destDir, "ffprobe"), 0755)
				logger.Info("FFprobe extraído para: %s", filepath.Join(destDir, "ffprobe"))
				
				// FFprobe baixado e extraído com sucesso
				archive.Close()
				return nil
			}
		}
		
		archive.Close()
	}
	
	return fmt.Errorf("falha ao baixar FFprobe: %v", lastError)
}

// Helpers

// Baixar arquivo com barra de progresso
func downloadFileWithProgress(ctx context.Context, url, destPath string, logger *Logger) error {
	// Criar diretório de destino se não existir
	err := os.MkdirAll(filepath.Dir(destPath), 0755)
	if err != nil {
		return fmt.Errorf("falha ao criar diretório: %v", err)
	}
	
	// Criar request com context para timeout e cancelamento
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar request: %v", err)
	}
	
	// Adicionar User-Agent para evitar 403 Forbidden
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	
	// Fazer request com cliente HTTP
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao fazer download: %v", err)
	}
	defer resp.Body.Close()
	
	// Verificar status code
	if resp.StatusCode != 200 {
		return fmt.Errorf("status code inválido: %d", resp.StatusCode)
	}
	
	// Criar arquivo de destino
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo: %v", err)
	}
	defer out.Close()
	
	// Configurar barra de progresso
	fileSize := resp.ContentLength
	
	progressOptions := ProgressTrackerOptions{
		Total:       fileSize,
		Description: "Baixando FFmpeg",
		Logger:      logger,
		ShowBytes:   true,
		ShowSpeed:   true,
	}
	
	progress := NewProgressTrackerWithOptions(progressOptions)
	
	// Configurar reader com progresso
	progressReader := &progressReader{
		reader:  resp.Body,
		tracker: progress,
		total:   fileSize,
	}
	
	// Copiar dados com buffer
	_, err = io.Copy(out, progressReader)
	if err != nil {
		return fmt.Errorf("erro ao salvar arquivo: %v", err)
	}
	
	progress.Finish()
	return nil
}

// Obter extensão do arquivo baseado na URL
func getArchiveExtension(url string) string {
	if strings.Contains(url, ".zip") {
		return ".zip"
	} else if strings.Contains(url, ".tar.xz") {
		return ".tar.xz"
	} else if strings.Contains(url, ".tar.gz") {
		return ".tar.gz"
	}
	
	// Default para zip
	return ".zip"
}

// Verificar instalação do FFmpeg
func testFFmpegInstallation(ffmpegPath, ffprobePath string) error {
	// Verificar se os executáveis existem
	if !fileExists(ffmpegPath) {
		return fmt.Errorf("executável do FFmpeg não encontrado: %s", ffmpegPath)
	}
	
	if !fileExists(ffprobePath) {
		return fmt.Errorf("executável do FFprobe não encontrado: %s", ffprobePath)
	}
	
	// Testar execução do FFmpeg
	cmd := exec.Command(ffmpegPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("erro ao executar FFmpeg: %v (output: %s)", err, string(output))
	}
	
	// Testar execução do FFprobe
	cmd = exec.Command(ffprobePath, "-version")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("erro ao executar FFprobe: %v (output: %s)", err, string(output))
	}
	
	return nil
}

// Verificar se um arquivo existe
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Obter tamanho de um arquivo
func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// Copiar um arquivo
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	return err
}

// Obter versão do FFmpeg
func getFFmpegVersion(ffmpegPath string) string {
	cmd := exec.Command(ffmpegPath, "-version")
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}
	
	// Ler a primeira linha, que contém a versão
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	if scanner.Scan() {
		line := scanner.Text()
		// Formato típico: "ffmpeg version 4.3.1 Copyright..."
		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[0] == "ffmpeg" && parts[1] == "version" {
			return parts[2]
		}
	}
	
	return "Unknown"
}

// Reader com progresso
type progressReader struct {
	reader  io.Reader
	tracker *ProgressTracker
	total   int64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.tracker.Update(pr.tracker.lastProgress + int64(n))
	}
	return n, err
} 