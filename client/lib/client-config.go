package splitpt_client

import (
	"errors"
	"log"

	"github.com/BurntSushi/toml"
)

type SplitPTConfig struct {
	SplittingAlg string
	Connections  map[string][]struct {
		BinPath   string
		Transport string
		Args      []string
		Cert      string
		Bridge    string
	}
}

func GetClientTOMLConfig(tomlFilename string) (*SplitPTConfig, error) {
	log.Printf("Decoding TOML")
	var config SplitPTConfig
	_, err := toml.DecodeFile(tomlFilename, &config)
	if err != nil {
		log.Printf("Error decoding TOML config")
		return nil, err
	}

	switch config.SplittingAlg {
	case "round-robin":
		log.Printf(config.SplittingAlg)
	case "random":
		log.Printf(config.SplittingAlg)
	default:
		log.Printf("Invalid splitting algorithm")
		return nil, errors.New("Invalid splitting algorithm in TOML")
	}
	log.Printf("%v connections found", len(config.Connections["connections"]))
	for _, conn := range config.Connections["connections"] {
		log.Printf("Connections: ")
		log.Printf(conn.Transport)
		log.Printf(conn.Cert)
		log.Printf(conn.Bridge)
	}
	return &config, nil
}
