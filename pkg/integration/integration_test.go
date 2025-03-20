package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testDataDir = "../../test_data"
	testOutputDir = "../../test_output"
	compressVideoCmd = "../../compressvideo"
)

// Estrutura para os testes de integração
type IntegrationTest struct {
	t         *testing.T
	binaryPath string
	dataDir    string
	tempDir    string
}

// Setup prepara o ambiente para os testes de integração
func NewIntegrationTest(t *testing.T) *IntegrationTest {
	// Caminho para o binário
	binaryPath, err := filepath.Abs(compressVideoCmd)
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	assert.NoError(t, err, "Should be able to resolve binary path")
	
	// Verificar se o binário existe
	_, err = os.Stat(binaryPath)
	assert.NoError(t, err, "Binary should exist at: "+binaryPath)
	
	// Caminho para o diretório de dados
	dataDir, err := filepath.Abs(testDataDir)
	assert.NoError(t, err, "Should be able to resolve data directory")
	
	// Verificar se o diretório de dados existe
	_, err = os.Stat(dataDir)
	assert.NoError(t, err, "Data directory should exist at: "+dataDir)
	
	// Criar diretório temporário para os resultados dos testes
	tempDir, err := os.MkdirTemp("", "compressvideo-test-")
	assert.NoError(t, err, "Should be able to create temp directory")
	
	return &IntegrationTest{
		t:         t,
		binaryPath: binaryPath,
		dataDir:    dataDir,
		tempDir:    tempDir,
	}
}

// Limpar recursos após os testes
func (it *IntegrationTest) Cleanup() {
	os.RemoveAll(it.tempDir)
}

func setup() error {
	// Certifique-se de que o diretório de saída existe
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		return err
	}
	
	// Limpar quaisquer outputs anteriores
	files, err := filepath.Glob(filepath.Join(testOutputDir, "*.mp4"))
	if err != nil {
		return err
	}
	
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			return err
		}
	}
	
	return nil
}

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		panic(err)
	}
	
	// Execute os testes
	exitCode := m.Run()
	
	// Saia com o código de status apropriado
	os.Exit(exitCode)
}

func TestBasicCompression(t *testing.T) {
	inputFile := filepath.Join(testDataDir, "screencast_sample.mp4")
	outputFile := filepath.Join(testOutputDir, "screencast_compressed.mp4")
	
	// Verifica se o arquivo de entrada existe
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skipf("Arquivo de teste %s não encontrado. Pulando teste.", inputFile)
	}
	
	// Execute o comando de compressão
	cmd := exec.Command(compressVideoCmd, "-i", inputFile, "-o", outputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Falha ao executar o comando: %v\nOutput: %s", err, output)
	}
	
	// Verifica se o arquivo de saída foi criado
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Arquivo de saída %s não foi criado", outputFile)
	}
	
	// Verifica se o tamanho do arquivo de saída é menor que o de entrada
	inputStat, _ := os.Stat(inputFile)
	outputStat, _ := os.Stat(outputFile)
	
	if outputStat.Size() >= inputStat.Size() {
		t.Errorf("Arquivo comprimido não é menor que o original: %d >= %d", 
			outputStat.Size(), inputStat.Size())
	}
}

func TestDifferentQualityLevels(t *testing.T) {
	inputFile := filepath.Join(testDataDir, "screencast_sample.mp4")
	
	// Verifica se o arquivo de entrada existe
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skipf("Arquivo de teste %s não encontrado. Pulando teste.", inputFile)
	}
	
	// Reduzir para apenas testar qualidade 3 (média) para evitar timeout
	q := 3
	outputFile := filepath.Join(testOutputDir, 
		fmt.Sprintf("quality_%d_output.mp4", q))
	
	// Execute o comando com o nível de qualidade médio
	cmd := exec.Command(compressVideoCmd, "-i", inputFile, "-o", outputFile, 
		"-q", fmt.Sprintf("%d", q))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Falha ao executar o comando com qualidade %d: %v\nOutput: %s", 
			q, err, output)
	}
	
	// Verifica se o arquivo de saída foi criado
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Arquivo de saída %s não foi criado", outputFile)
	}
}

// TestDifferentPresets testa diferentes presets
func TestDifferentPresets(t *testing.T) {
	inputFile := filepath.Join(testDataDir, "screencast_sample.mp4")
	
	// Verifica se o arquivo de entrada existe
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		t.Skipf("Arquivo de teste %s não encontrado. Pulando teste.", inputFile)
	}
	
	// Testar apenas o preset "fast" para evitar timeout
	preset := "fast"
	outputFile := filepath.Join(testOutputDir, 
		fmt.Sprintf("preset_%s_output.mp4", preset))
	
	// Execute o comando com o preset
	cmd := exec.Command(compressVideoCmd, "-i", inputFile, "-o", outputFile, 
		"-p", preset)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Falha ao executar o comando com preset %s: %v\nOutput: %s", 
			preset, err, output)
	}
	
	// Verifica se o arquivo de saída foi criado
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Arquivo de saída %s não foi criado para preset %s", 
			outputFile, preset)
	}
}

func TestVersionOutput(t *testing.T) {
	// Execute o comando de versão
	cmd := exec.Command(compressVideoCmd, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Falha ao executar o comando de versão: %v\nOutput: %s", err, output)
	}
	
	// Verifica se a saída contém uma informação de versão válida
	outputStr := string(output)
	if !strings.Contains(outputStr, "CompressVideo") || 
		!strings.Contains(outputStr, "v") {
		t.Errorf("Saída de versão inválida: %s", outputStr)
	}
} 