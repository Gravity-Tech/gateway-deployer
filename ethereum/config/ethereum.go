package config

import "fmt"

type EthereumConfig struct {
	GravityBftCoefficient int
	NodeUrl               string
	PrivKey               string
	ConsulsAddress        []string
	ExistingGravityAddress string
	ExistingTokenAddress  string
}


func (cfg *EthereumConfig) Validate() error {
	if cfg.ExistingGravityAddress == "" {
		return fmt.Errorf("gravity address is empty")
	}
	if cfg.ExistingTokenAddress == "" {
		return fmt.Errorf("existing token address is empty")
	}
	if len(cfg.ConsulsAddress) == 0 {
		return fmt.Errorf("consuls list is empty")
	}
	if cfg.NodeUrl == "" {
		return fmt.Errorf("node url is empty")
	}
	if cfg.PrivKey == "" {
		return fmt.Errorf("priv key of deployer is empty")
	}
	if cfg.GravityBftCoefficient <= 0 {
		return fmt.Errorf("bft coefficient cannot be less than 1")
	}

	return nil
}