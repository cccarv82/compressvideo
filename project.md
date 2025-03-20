Prompt para Desenvolvimento do CompressVideo: Ferramenta de Compressão Inteligente de Vídeo em Go
Visão Geral da Ferramenta
Desenvolva uma aplicação CLI em Go chamada compressvideo que oferece compressão inteligente de vídeos, reduzindo o tamanho dos arquivos em até 70% enquanto mantém a maior qualidade visual possível. A ferramenta deve ser instalável via go install github.com/cccarv82/compressvideo@latest, ter interface simples via linha de comando, usar flags para configuração, e mostrar o progresso da compressão em tempo real.
Backlog do Produto
Divida o desenvolvimento nas seguintes histórias de usuário (sprints pequenos):
Sprint 1: Fundação e CLI

Configuração inicial do projeto

Criar estrutura de diretórios Go padrão
Configurar módulo Go e dependências iniciais
Estabelecer convenções de código e documentação

Interface de linha de comando

Implementar parser de flags com cobra/urfave-cli
Criar help detalhado com exemplos (-h)
Validar entradas do usuário

Estrutura base de execução

Criar pipeline básica de processamento
Implementar logging e feedback ao usuário
Estruturar tratamento de erros

Sprint 2: Análise de Vídeo

Integração com FFmpeg

Estabelecer bindings com FFmpeg
Criar funções para extrair metadados de vídeos
Implementar detecção de formato e codecs

Analisador de conteúdo

Desenvolver algoritmo para classificar tipo de vídeo
Analisar complexidade de movimento e cenas
Detectar características especiais (animação, screencast, etc.)

Estimativa de compressão

Calcular potencial de compressão
Determinar parâmetros iniciais baseados na análise
Criar mecanismo de feedback para resultados

Sprint 3: Motor de Compressão Inteligente

Sistema de decisão de codec

Implementar seleção automática de codec ideal (H.265, AV1, VP9)
Ajustar parâmetros específicos por codec
Otimizar para diferentes tipos de conteúdo

Algoritmo de compressão adaptativa

Desenvolver sistema que balanceia qualidade/tamanho
Implementar ajuste dinâmico de bitrate por cena
Criar presets inteligentes baseados no conteúdo

Processamento paralelo

Implementar segmentação de vídeo para processamento
Utilizar goroutines para compressão paralela
Otimizar uso de recursos do sistema

Sprint 4: Interface e Experiência do Usuário

Barra de progresso e interface

Criar visualização em tempo real do progresso
Mostrar estatísticas durante compressão
Implementar estimativa de tempo restante

Geração de relatório

Criar resumo final de compressão
Mostrar comparativo antes/depois
Oferecer dicas baseadas no resultado

Instalação e distribuição

Configurar build multiplataforma
Preparar para distribuição via go install
Criar documentação de instalação e uso

Sprint 5: Finalização e Testes

Testes e benchmark

Criar suite de testes automatizados
Realizar benchmark com diferentes tipos de vídeos
Otimizar performance

Documentação completa

Escrever documentação técnica
Criar exemplos e casos de uso
Documentar todas as flags disponíveis

Polimento e lançamento

Revisão final de código
Preparar release inicial
Configurar CI/CD para builds automáticos

Especificação da CLI
A ferramenta deve suportar as seguintes flags:
Copycompressvideo -i input.mp4 [-o output.mp4] [-q quality] [-p preset] [-f force] [-v verbose]
Onde:

-i, --input: Caminho do arquivo de vídeo a ser comprimido (obrigatório)
-o, --output: Caminho para salvar o arquivo comprimido (opcional, usa mesmo nome com sufixo se omitido)
-q, --quality: Nível de qualidade de 1-5 (1=máxima compressão, 5=máxima qualidade, padrão=3)
-p, --preset: Preset de compressão ("fast", "balanced", "thorough", padrão="balanced")
-f, --force: Sobrescrever arquivo de saída se existir
-v, --verbose: Mostrar informações detalhadas durante o processo
-h, --help: Mostrar ajuda detalhada

Comportamento da Interface

Exibição Inicial:

Mostrar detector de tipo de conteúdo
Exibir parâmetros selecionados automaticamente
Apresentar estimativa de redução de tamanho

Durante a Compressão:

Barra de progresso com porcentagem
Tempo estimado restante
Taxa de compressão atual

Ao Finalizar:

Comparativo de tamanho antes/depois
Tempo total de processamento
Taxa de compressão final
Sugestões baseadas no resultado

Diretrizes Técnicas

Foco em código limpo e bem documentado
Tratamento robusto de erros
Testes unitários e de integração
Utilização eficiente de goroutines para processamento paralelo
Capacidade de instalação via go install
Binários autocontidos (incluir dependências necessárias)

Esta aplicação deve fazer o mínimo de perguntas ao usuário, utilizando inteligência para determinar os melhores parâmetros com base no tipo de vídeo e contexto, mantendo a interface simples e o processo o mais automatizado possível.
