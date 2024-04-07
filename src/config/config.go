package config

import (
	"gopkg.in/yaml.v2"
	"log"
	"os"
)

type Config struct {
	Config struct {
		NThreads int `yaml:"threads"`
	} `yaml:"config"`

	Redis struct {
		Addr string `yaml:"addr"`
		User string `yaml:"user"`
		Pass string `yaml:"pass"`
		Db   string `yaml:"db"`
	} `yaml:"redis"`
}

func ReadConfig() *Config {
	cfg := &Config{}
	f, err := os.Open("config.yml")
	if err != nil {
		log.Fatalf("Imposible procesar o no existe:  config.yml \n")
	}
	//defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		log.Fatalf("Imposible procesar archivo config: ", err)
	}

	return cfg
}
