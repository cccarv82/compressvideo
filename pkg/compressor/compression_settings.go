package compressor

import (
	"fmt"
)

// CompressionSettings contém as configurações para compressão de vídeo
type CompressionSettings struct {
	// Codec de vídeo a ser usado (h264, hevc/h265, etc.)
	Codec string
	
	// CRF (Constant Rate Factor) - controla a qualidade
	// Valores mais baixos = melhor qualidade, maior arquivo
	// h264: 18-28 (23 é padrão)
	// h265: 23-28 (28 é padrão)
	CRF int
	
	// Preset de codificação (ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow)
	// Presets mais lentos = melhor compressão
	Preset string
	
	// Bitrate alvo em bits por segundo (se especificado, substitui o CRF)
	Bitrate int64
	
	// Codec de áudio a ser usado (aac, opus, etc.)
	AudioCodec string
	
	// Bitrate de áudio em bits por segundo
	AudioBitrate int64
	
	// Ajustes específicos do codec (ex: x264-params, x265-params)
	CodecParams string
	
	// Parâmetros adicionais para o FFmpeg
	ExtraParams map[string]string
}

// NewCompressionSettings cria uma nova instância de configurações de compressão com valores padrão
func NewCompressionSettings() CompressionSettings {
	return CompressionSettings{
		Codec:       "h264",
		CRF:         23,
		Preset:      "medium",
		AudioCodec:  "aac",
		AudioBitrate: 128000,
		ExtraParams: make(map[string]string),
	}
}

// FromMap converte um mapa de strings para uma estrutura CompressionSettings
func FromMap(settings map[string]string) CompressionSettings {
	cs := NewCompressionSettings()
	
	if codec, ok := settings["codec"]; ok {
		cs.Codec = codec
	}
	
	if crf, ok := settings["crf"]; ok {
		// Ignora erro de conversão e mantém o padrão
		if val, err := parseInt(crf); err == nil {
			cs.CRF = val
		}
	}
	
	if preset, ok := settings["preset"]; ok {
		cs.Preset = preset
	}
	
	if bitrate, ok := settings["bitrate"]; ok {
		// Ignora erro de conversão e mantém o padrão
		if val, err := parseInt64(bitrate); err == nil {
			cs.Bitrate = val
		}
	}
	
	if audioCodec, ok := settings["audio_codec"]; ok {
		cs.AudioCodec = audioCodec
	}
	
	if audioBitrate, ok := settings["audio_bitrate"]; ok {
		// Ignora erro de conversão e mantém o padrão
		if val, err := parseInt64(audioBitrate); err == nil {
			cs.AudioBitrate = val
		}
	}
	
	if codecParams, ok := settings["codec_params"]; ok {
		cs.CodecParams = codecParams
	}
	
	// Copia outros parâmetros para ExtraParams
	for k, v := range settings {
		switch k {
		case "codec", "crf", "preset", "bitrate", "audio_codec", "audio_bitrate", "codec_params":
			// Já processados acima
		default:
			cs.ExtraParams[k] = v
		}
	}
	
	return cs
}

// ToMap converte uma estrutura CompressionSettings para um mapa de strings
func (cs CompressionSettings) ToMap() map[string]string {
	result := make(map[string]string)
	
	result["codec"] = cs.Codec
	result["crf"] = formatInt(cs.CRF)
	result["preset"] = cs.Preset
	
	if cs.Bitrate > 0 {
		result["bitrate"] = formatInt64(cs.Bitrate)
	}
	
	result["audio_codec"] = cs.AudioCodec
	result["audio_bitrate"] = formatInt64(cs.AudioBitrate)
	
	if cs.CodecParams != "" {
		result["codec_params"] = cs.CodecParams
	}
	
	// Copia parâmetros extras
	for k, v := range cs.ExtraParams {
		result[k] = v
	}
	
	return result
}

// Funções auxiliares para conversão

func parseInt(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

func parseInt64(s string) (int64, error) {
	var v int64
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}

func formatInt64(i int64) string {
	return fmt.Sprintf("%d", i)
} 