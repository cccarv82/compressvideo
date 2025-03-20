# Changelog

## [1.2.0] - Robustez e Confiabilidade

### Adicionado
- Comando `repair-ffmpeg` para consertar instalações problemáticas do FFmpeg
- Suporte melhorado para sistemas Windows, incluindo tratamento especial para caminhos com espaços
- Múltiplas fontes de download para o FFmpeg em caso de falha
- Sistema de reparo automático para detecção de problemas com FFmpeg

### Melhorado
- Tratamento avançado de erros no FFmpeg com mensagens mais informativas
- Melhor detecção de problemas específicos do Windows
- Verificação de integridade dos arquivos baixados
- Sistema de tentativas múltiplas para download e extração
- Manipulação mais robusta de diferentes formatos de arquivo (ZIP, TAR.GZ, TAR.XZ)
- Interface de usuário com mensagens mais claras sobre problemas e soluções

## [1.1.0] - Aprimoramento de Usabilidade

### Adicionado
- Sistema de download automático do FFmpeg quando não está instalado
- Verificação inteligente de disponibilidade do FFmpeg
- Armazenamento da versão baixada do FFmpeg para uso futuro
- Suporte à execução sem dependências externas
- Mensagens amigáveis durante a detecção e download do FFmpeg

### Melhorado
- Documentação expandida para incluir informações sobre a instalação automática do FFmpeg
- Interface de inicialização mais robusta
- Melhor tratamento de ambientes sem FFmpeg instalado

## [1.0.0] - Sprint 5 - Testes e Finalização

### Adicionado
- Testes de integração completos para verificar o fluxo da aplicação
- Testes de desempenho (benchmarks) para componentes críticos
- Suporte aprimorado para diferentes sistemas operacionais
- Utilitários específicos por sistema operacional
- Exportação de APIs para facilitar testes e extensibilidade
- Preparação para empacotamento e distribuição

### Melhorado
- Documentação expandida e revisada
- Maior cobertura de testes
- Otimizações de desempenho baseadas em benchmarks
- Estabilidade em diferentes ambientes
- Tratamento consistente de caminhos de arquivos
- Versão atualizada para 1.0.0 para lançamento de produção

## [0.4.0] - Sprint 4 - Correções e Melhorias

### Corrigido
- Correção no arquivo main.go para compatibilidade com a estrutura de comandos
- Ajustes nas referências a constantes de complexidade de movimento
- Validação aprimorada das configurações antes da compressão
- Resolução de problemas na interface entre os componentes de análise e compressão
- Corrigidos vários problemas de tipos e inconsistências nas APIs

### Melhorado
- Atualização da documentação para incluir informações sobre o Sprint 4
- Maior estabilidade na compilação e execução
- Melhoria na detecção de erros e nas mensagens de feedback ao usuário
- Código refatorado para melhor legibilidade e manutenção

## [0.3.0] - Sprint 3 - Interface e Experiência do Usuário

### Adicionado
- Sistema avançado de relatórios com detalhes completos da compressão
- Geração de relatório em arquivo de texto para cada compressão
- Pontuação de desempenho da compressão (0-100)
- Estimativa de qualidade visual com descrições significativas
- Dicas personalizadas de otimização baseadas no resultado da compressão
- Cálculo de tempo economizado em transferências
- Interface colorida no terminal com suporte a emojis
- Barra de progresso aprimorada com estimativas de tempo restante
- Formatação amigável de tamanhos de arquivo e taxas de bits
- Indicadores visuais para tipos de conteúdo e complexidade de movimento

### Melhorado
- Logger com cores ANSI e diferentes níveis de informação
- Exibição de informações com seções e formatação clara
- Estrutura de comandos para exibição mais organizada
- Makefile com suporte a compilação multiplataforma
- Sistema de empacotamento para distribuição em diferentes sistemas
- Instalação simplificada via `go install`
- Documentação para desenvolvedores e usuários finais

## [0.2.0] - Sprint 3 - Motor de Compressão Inteligente

### Adicionado
- Motor de compressão efetiva para vídeos
- Implementação do processamento paralelo usando goroutines
- Segmentação de vídeos para compressão paralela
- Suporte aos codecs H.264 e H.265
- Seleção automática de codec baseada no tipo de conteúdo
- Ajuste dinâmico de parâmetros de compressão
- Presets de qualidade (1-5) que balanceiam qualidade e tamanho
- Presets de velocidade (fast, balanced, thorough)
- Medição de progresso em tempo real
- Relatório de resultados da compressão

## [0.1.1] - Melhorias nos Testes e Documentação

### Adicionado
- Script de download de vídeos de teste em `scripts/download_test_videos.py`
- Pasta `data/` para armazenar vídeos de teste (excluída do controle de versão)
- Comando `make download-test-videos` para facilitar o download
- Documentação expandida sobre recursos de teste

### Melhorado
- Documentação de recursos para desenvolvedores
- Organização do projeto com diretório dedicado para dados de teste

## [0.1.0] - Sprint 2 - Análise de Vídeo

### Adicionado
- Integração com FFmpeg para análise de vídeos
- Detecção automática de tipo de conteúdo (screencast, animação, jogos, ação ao vivo, etc.)
- Análise de complexidade de movimento
- Detecção de mudanças de cena
- Análise de complexidade de quadros
- Seleção inteligente de codec com base no tipo de conteúdo
- Cálculo de bitrate ideal para qualidade desejada
- Estimativa de potencial de compressão
- Geração de configurações de compressão otimizadas
- Testes unitários para o analisador de conteúdo
- Makefile para facilitar o build e teste

## [0.0.1] - Sprint 1 - Fundação e CLI

### Adicionado
- Estrutura inicial do projeto
- Interface de linha de comando usando cobra
- Validação de entradas do usuário
- Sistema de logging
- Barra de progresso
- Estruturas básicas para manipulação de arquivos de vídeo
- Documentação inicial no README.md 