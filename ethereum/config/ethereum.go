package config

type EthereumConfig struct {
	GravityBftCoefficient int
	NodeUrl               string
	PrivKey               string
	ConsulsAddress        [5]string
	ExistingTokenAddress  string
	//NewERC20              NewERC20
}

//type NewERC20 struct {
//	Name                string
//	Symbol              string
//	TokenOwnerForLUPort string
//}
