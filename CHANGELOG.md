# Changelog

## [1.5.6] - Remoção Completa de Aceleração de Hardware

### Removido
- Remoção completa do código de aceleração de hardware
- Eliminados todos os tratamentos específicos para NVIDIA, Intel e AMD
- Removidas estruturas de dados e campos relacionados à aceleração via GPU

### Melhorado
- Código simplificado para utilizar apenas codecs baseados em CPU
- Processo de compressão mais estável e previsível
- Melhor desempenho geral sem tentativas de fallback para GPU

## [1.5.5] - Simplificação e Remoção de Aceleração de Hardware

### Removido
- Funcionalidade de aceleração de hardware (GPU) removida devido a problemas persistentes de compatibilidade
- Opção de linha de comando `-a/--hwaccel` removida
- Eliminado código relacionado à detecção e utilização de aceleradores NVIDIA, Intel e AMD

### Melhorado
- Interface simplificada sem opções que causavam erros frequentes
- Melhor foco no processamento estável via CPU
- Mensagens de erro menos confusas para o usuário final

## [1.5.4] - Correções de Compatibilidade NVENC e Sintaxe

### Corrigido
- Erro de sintaxe no tratamento de erros relacionados à aceleração de hardware
- Corrigidas referências aos campos de altura de vídeo para compatibilidade
- Melhorado o tratamento de erros específicos do NVENC no Windows
- Parâmetros NVENC ajustados para maior estabilidade

### Melhorado
- Detecção de resolução de vídeo para ajuste automático de bitrate
- Implementação mais robusta do fallback de GPU para CPU
- Tratamento de erros com mensagens mais detalhadas

## [1.5.3] - Correções específicas para NVIDIA no Windows

### Corrigido
- Erro 0xffffffea em sistemas Windows ao usar aceleração NVIDIA
- Problema de inicialização do encoder NVENC corrigido com parâmetros simplificados
- Implementação de fallback automático para codificação via CPU quando códigos de erro específicos são detectados
- Tratamento de erros específicos do Windows relacionados à aceleração de hardware

### Melhorado
- Parâmetros de codificação NVENC optimizados para Windows
- Configuração simplificada para evitar conflitos de parâmetros no Windows
- Sistema de atribuição automática de bitrate quando não especificado para encoders de hardware
- Melhor detecção e tratamento de erros específicos da plataforma Windows com GPUs NVIDIA

## [1.5.2] - Correções para Aceleração de Hardware no Windows

### Corrigido
- Problema de compatibilidade com aceleração NVIDIA em sistemas Windows
- Adicionado fallback automático para CPU quando a aceleração de hardware falha
- Tratamento especial para erros específicos de aceleração de hardware
- Melhorias nos parâmetros de codificação NVENC para maior compatibilidade

### Melhorado
- Detecção e manipulação de erros durante o processo de compressão
- Feedback mais detalhado ao usuário sobre problemas de aceleração
- Configurações específicas por plataforma para melhor compatibilidade

## [1.5.1] - Melhorias na Aceleração de Hardware

### Adicionado
- Mensagens informativas detalhadas sobre qual acelerador de hardware está sendo usado
- Detecção e seleção automática aprimorada do melhor acelerador disponível
- Indicação visual de aceleração de hardware ativa nos relatórios de compressão
- Feedback em tempo real sobre o processo de aceleração

### Melhorado
- Interface de usuário para mostrar claramente quando a aceleração de hardware está ativa
- Conversão automática de codecs para versões aceleradas por hardware
- Detecção mais robusta de GPUs NVIDIA, Intel e AMD
- Mensagens de diagnóstico quando aceleradores específicos não estão disponíveis

## [1.5.0] - Cache de Análise de Conteúdo

### Adicionado
- Sistema de cache para armazenar resultados de análise de vídeos
- Banco de dados SQLite para armazenar os dados de forma eficiente
- Reutilização de configurações para vídeos semelhantes
- Comando `cache` para gerenciar o cache de análise
- Opções de linha de comando para controlar o comportamento do cache:
  - `--use-cache/-c`: Ativar o uso do cache
  - `--clear-cache/-C`: Limpar entradas expiradas do cache
  - `--cache-max-age/-A`: Definir idade máxima das entradas do cache em dias

### Melhorado
- Tempo de análise reduzido para vídeos previamente processados
- Processamento de lotes de vídeos mais rápido
- Detecção de vídeos similares para compartilhar configurações
- Gerenciamento inteligente do cache com limpeza automática
- Processamento em diretórios beneficiado pelo reuso de configurações

## [1.4.0] - Aceleração de Hardware (GPU)

### Adicionado
- Suporte para codificação por hardware usando GPUs
- Aceleração NVIDIA (NVENC) para placas GeForce/Quadro
- Aceleração Intel QuickSync (QSV) para processadores Intel com GPU integrada
- Aceleração AMD (AMF) para placas Radeon
- Detecção automática de aceleradores disponíveis no sistema
- Nova opção de linha de comando `-a/--hwaccel` para selecionar acelerador
- Detecção automática dos melhores parâmetros para cada tipo de GPU

### Melhorado
- Velocidade de compressão até 10x mais rápida com aceleração de hardware
- Ajuste automático de bitrate e configurações de qualidade para codecs de hardware
- Documentação sobre uso de GPUs para compressão de vídeo
- Corrigido cálculo de proporção para evitar largura zero ao usar resolução 720p
- Ajustado comportamento do preset "thorough" para evitar conflitos de configurações
- Melhorada lógica de escala para qualidade alta e máxima

## [1.3.7] - Correção da Interface de Progresso e Cálculo de Tamanho

### Corrigido
- Restaurada a barra de progresso durante a compressão de vídeo
- Corrigido cálculo de tamanhos de arquivo original e final
- Melhorada exibição de porcentagem de redução de tamanho
- Interface de usuário volta a exibir informações detalhadas durante o processo
- Mantida a correção para o erro "Stderr already set"

## [1.3.6] - Correção para Erros de Execução do FFmpeg

### Corrigido
- Corrigido erro "exec: Stderr already set" ao executar o FFmpeg
- Melhorada a forma de capturar a saída padrão e de erro do FFmpeg
- Garantida a utilização correta do caminho do executável FFmpeg
- Eliminados conflitos na captura dos streams de saída do processo

## [1.3.5] - Correção Adicional para o Processamento de Vídeo

### Corrigido
- Resolvido problema persistente com erro "exit status 0xabafb00" ao usar preset "thorough" e qualidade 1
- Melhorado o tratamento de filtros de escala para evitar configurações inválidas no FFmpeg
- Corrigido formato dos comandos FFmpeg para garantir compatibilidade em diferentes cenários
- Adicionado log detalhado para facilitar diagnóstico de erros no processamento de vídeo

## [1.3.4] - Correção de Bug nas Configurações de Redimensionamento

### Corrigido
- Resolvido erro de compressão quando usando preset "thorough" com qualidade 1 ou 2
- Corrigido cálculo de largura proporcional para vídeos redimensionados, evitando largura igual a zero
- Ajustada a lógica de aplicação de bitrate vs. CRF para presets de alta qualidade
- Melhorada a compatibilidade com diferentes proporções de aspecto de vídeo

## [1.3.3] - Melhorias na Interface de Usuário

### Melhorado
- Informações mais claras sobre o bitrate durante a compressão
- Mensagens mais informativas sobre arquivos que serão ignorados
- Melhor feedback sobre os modos de compressão (CRF vs. Bitrate fixo)
- Exibição aprimorada do progresso de diretórios com múltiplos arquivos

## [1.3.2] - Otimização no Processamento de Diretórios

### Melhorado
- Lógica aprimorada para ignorar arquivos que já possuem uma versão comprimida correspondente
- Verificação inteligente de arquivos já processados em diretórios
- Redução de processamento desnecessário durante a compressão em lote

## [1.3.1] - Processamento Recursivo de Diretórios

### Adicionado
- Flag `-r, --recursive` para processar subdiretórios recursivamente
- Processamento automático de todos os vídeos em subdiretórios aninhados
- Suporte para estruturas de diretórios complexas
- Preservação da estrutura de diretório na saída

### Melhorado
- Feedback detalhado durante o processamento recursivo
- Contabilização de arquivos melhorada para diretórios aninhados
- Documentação atualizada incluindo a nova flag

## [1.3.0] - Suporte a Processamento de Diretórios

### Adicionado
- Suporte para processamento de todos os vídeos em um diretório quando um diretório é fornecido como entrada
- Detecção automática de arquivos de vídeo em um diretório
- Ignorar automaticamente arquivos que já terminam com "-compressed"
- Suporte para especificar um diretório de saída diferente

### Melhorado
- Feedback detalhado durante o processamento de múltiplos arquivos
- Estatísticas de processamento, incluindo contagem de arquivos processados e ignorados
- Tratamento robusto de caminhos em diferentes sistemas operacionais

## [1.2.4] - Aprimoramento da Barra de Progresso

### Corrigido
- Corrigido o problema da barra de progresso que não atualizava corretamente
- Implementado suporte adequado a códigos ANSI para atualização na mesma linha
- Removidas mensagens de debug que interferiam na visualização da barra de progresso

### Melhorado
- Barra de progresso agora atualiza na mesma linha para uma interface mais limpa
- Melhor feedback visual durante o processo de compressão
- Experiência de usuário aprimorada durante operações de longa duração

## [1.2.3] - Melhorias na Barra de Progresso

### Corrigido
- Correção no cálculo de progresso para arquivos grandes
- Melhor tratamento de progresso para compressão paralela de segmentos
- Limitação na frequência de atualizações da barra de progresso para evitar sobrecarga
- Combinação eficiente de progress ID e valor em um único inteiro

### Melhorado
- Interface de progresso mais responsiva com arquivos grandes
- Menor sobrecarga de CPU durante a exibição de progresso
- Melhor feedback visual durante operações de longa duração

## [1.2.2] - Robustez e Confiabilidade

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