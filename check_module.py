try:
    import whisper_ctranslate2
    print("✅ whisper_ctranslate2 importado com sucesso!")
    print(f"Versão: {whisper_ctranslate2.__version__}")
    print(f"Caminho: {whisper_ctranslate2.__file__}")
except ImportError as e:
    print(f"❌ Erro ao importar whisper_ctranslate2: {e}")

# Verificar outras dependências relevantes
dependencies = [
    "numpy", "tqdm", "ctranslate2", "faster_whisper", 
    "sounddevice", "huggingface_hub", "tokenizers", 
    "onnxruntime", "av"
]

print("\nVerificando dependências:")
for dep in dependencies:
    try:
        module = __import__(dep)
        print(f"✅ {dep} encontrado")
    except ImportError as e:
        print(f"❌ {dep} não encontrado: {e}")

# Verificar caminhos Python
import sys
print("\nCaminhos Python (sys.path):")
for path in sys.path:
    print(f"- {path}")

# Verificar informações do sistema
import platform
print("\nInformações do sistema:")
print(f"Sistema: {platform.system()} {platform.release()}")
print(f"Versão Python: {platform.python_version()}")
print(f"Arquitetura: {platform.architecture()}") 