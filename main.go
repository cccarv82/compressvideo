package main

import (
	"github.com/cccarv82/compressvideo/cmd/compressvideo/cmd"
	// Importações para garantir que todos os comandos sejam inicializados
	_ "github.com/cccarv82/compressvideo/pkg/transcriber"
)

func main() {
	cmd.Execute()
} 