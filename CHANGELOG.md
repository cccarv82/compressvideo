# Changelog

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