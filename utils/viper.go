package utils

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config reúne as chaves de configuração lidas do .env e/ou do ambiente.
type Config struct {
	GEMINI_API_KEY string `mapstructure:"GEMINI_API_KEY"`
	ITEMS_PATH     string `mapstructure:"ITEMS_PATH"`
	PHONE_NUMBER   string `mapstructure:"PHONE_NUMBER"`
	CHROME_PATH    string `mapstructure:"CHROME_PATH"`
	DEFAULT_CITY   string `mapstructure:"DEFAULT_CITY"`
	PLANILHA_PATH  string `mapstructure:"PLANILHA_PATH"`
}

// chavesConhecidas lista todas as chaves que o Unmarshal deve observar.
var chavesConhecidas = []string{
	"GEMINI_API_KEY",
	"ITEMS_PATH",
	"PHONE_NUMBER",
	"CHROME_PATH",
	"DEFAULT_CITY",
	"PLANILHA_PATH",
}

// LoadConfig carrega a configuração a partir do arquivo .env em dir.
// Variáveis de ambiente reais têm precedência sobre o arquivo.
// Arquivo inexistente não é tratado como erro.
func LoadConfig(dir string) (Config, error) {
	var cfg Config

	v := viper.New()
	v.SetConfigType("env")
	v.SetConfigName(".env")
	v.AddConfigPath(dir)

	// Registra defaults vazios para que AutomaticEnv seja enxergado
	// pelo Unmarshal (viper só resolve env para chaves conhecidas).
	for _, k := range chavesConhecidas {
		v.SetDefault(k, "")
	}
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var naoEncontrado viper.ConfigFileNotFoundError
		if !errors.As(err, &naoEncontrado) {
			// Se o arquivo existe mas não pôde ser lido/interpretado,
			// devolve o erro real.
			if _, statErr := os.Stat(filepath.Join(dir, ".env")); statErr == nil {
				return Config{}, err
			}
		}
		// Arquivo ausente: segue apenas com o ambiente.
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
