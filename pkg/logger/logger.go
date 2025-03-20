package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel define o nível de logging
type LogLevel int

const (
	// LogLevelError só registra erros
	LogLevelError LogLevel = iota
	// LogLevelInfo registra informações gerais e erros
	LogLevelInfo
	// LogLevelDebug registra informações de debug, info e erros
	LogLevelDebug
)

// Códigos de cores ANSI
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[37m"
	colorWhite   = "\033[97m"
	
	colorBold      = "\033[1m"
	colorUnderline = "\033[4m"
)

// Logger gerencia logging e feedback ao usuário
type Logger struct {
	Level       LogLevel
	Verbose     bool
	UseColors   bool
	TimeFormat  string
	ShowLogTime bool
}

// NewLogger cria um novo Logger com o nível de log e verbosidade especificados
func NewLogger(verbose bool, useColors bool) *Logger {
	level := LogLevelInfo
	if verbose {
		level = LogLevelDebug
	}
	
	// Detecta se devemos usar cores (desabilita no Windows, exceto em ANSICON/ConEmu/WSL)
	if !useColors {
		useColors = true
		if runtime.GOOS == "windows" {
			// Verifica terminais Windows comuns que suportam ANSI
			_, hasAnsiCon := os.LookupEnv("ANSICON")
			_, hasConEmu := os.LookupEnv("ConEmuANSI")
			_, hasWT := os.LookupEnv("WT_SESSION")
			
			// Desabilita cores por padrão no Windows a menos que esteja em um terminal compatível
			useColors = hasAnsiCon || hasConEmu || hasWT
		}
	}
	
	return &Logger{
		Level:       level,
		Verbose:     verbose,
		UseColors:   useColors,
		TimeFormat:  "15:04:05",
		ShowLogTime: true,
	}
}

// formatMessage cria uma mensagem de log formatada com timestamp e nível
func (l *Logger) formatMessage(level, color, message string) string {
	var timePrefix string
	if l.ShowLogTime {
		timestamp := time.Now().Format(l.TimeFormat)
		timePrefix = fmt.Sprintf("[%s] ", timestamp)
	}
	
	if l.UseColors {
		return fmt.Sprintf("%s%s%s%s: %s%s", 
			timePrefix, color, level, colorReset, color, message)
	}
	
	return fmt.Sprintf("%s%s: %s", timePrefix, level, message)
}

// colorize adiciona cor a uma string se as cores estiverem habilitadas
func (l *Logger) colorize(color, message string) string {
	if l.UseColors {
		return color + message + colorReset
	}
	return message
}

// Info registra mensagens informativas
func (l *Logger) Info(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		fmt.Println(l.formatMessage("INFO", colorBlue, message))
	}
}

// Debug registra mensagens de debug
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.Level >= LogLevelDebug {
		message := fmt.Sprintf(format, args...)
		fmt.Println(l.formatMessage("DEBUG", colorCyan, message))
	}
}

// Error registra mensagens de erro
func (l *Logger) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, l.formatMessage("ERROR", colorRed, message))
}

// Warning registra mensagens de aviso
func (l *Logger) Warning(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		fmt.Println(l.formatMessage("WARNING", colorYellow, message))
	}
}

// Fatal registra uma mensagem de erro fatal e encerra o programa
func (l *Logger) Fatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, l.formatMessage("FATAL", colorRed+colorBold, message))
	os.Exit(1)
}

// Success registra uma mensagem de sucesso
func (l *Logger) Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Println(l.formatMessage("SUCCESS", colorGreen, message))
}

// Title exibe um título em negrito destacado
func (l *Logger) Title(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		
		// Cria uma linha de sinais de igual do mesmo tamanho da mensagem
		lineLength := len(message)
		line := strings.Repeat("=", lineLength)
		
		if l.UseColors {
			fmt.Println(l.colorize(colorYellow+colorBold, line))
			fmt.Println(l.colorize(colorYellow+colorBold, message))
			fmt.Println(l.colorize(colorYellow+colorBold, line))
		} else {
			fmt.Println(line)
			fmt.Println(message)
			fmt.Println(line)
		}
	}
}

// Section exibe um cabeçalho de seção
func (l *Logger) Section(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		
		if l.UseColors {
			fmt.Println(l.colorize(colorMagenta+colorBold, message))
			fmt.Println(l.colorize(colorMagenta, strings.Repeat("-", len(message))))
		} else {
			fmt.Println(message)
			fmt.Println(strings.Repeat("-", len(message)))
		}
	}
}

// Field registra um campo rotulado com valor
func (l *Logger) Field(label, format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		value := fmt.Sprintf(format, args...)
		
		if l.UseColors {
			fmt.Printf("%s: %s\n", 
				l.colorize(colorYellow, label),
				value)
		} else {
			fmt.Printf("%s: %s\n", label, value)
		}
	}
}

// Progress registra uma mensagem de progresso
func (l *Logger) Progress(format string, args ...interface{}) {
	if l.Level >= LogLevelInfo {
		message := fmt.Sprintf(format, args...)
		fmt.Println(l.formatMessage("PROGRESS", colorMagenta, message))
	}
}

// SetUseColors habilita ou desabilita a saída colorida
func (l *Logger) SetUseColors(useColors bool) {
	l.UseColors = useColors
}

// SetLevel define o nível de log
func (l *Logger) SetLevel(level LogLevel) {
	l.Level = level
} 