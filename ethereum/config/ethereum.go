package config

type EthereumConfig struct {
	GravityBftCoefficient int
	NodeUrl               string
	PrivKey               string
	ConsulsAddress        [5]string
	NewERC20              NewERC20
	//	ExistERC20Address *string TODO: Add exist token to LUPort
}

type NewERC20 struct {
	Name                string
	Symbol              string
	TokenOwnerForLUPort string
}
