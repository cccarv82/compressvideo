package cmd

import (
	// Importações necessárias para garantir que todos os pacotes sejam carregados
	_ "github.com/cccarv82/compressvideo/pkg/transcriber"
)

// Este arquivo garante que todos os pacotes necessários são importados,
// mesmo que não sejam diretamente utilizados em outros lugares.
// Isso ajuda a garantir que as funções init() de todos os pacotes
// são executadas durante a inicialização. 