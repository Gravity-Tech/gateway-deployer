package cmd

import (
	"fmt"
	"os"
	"github.com/Gravity-Tech/gateway-deployer/ethereum/config"
	"github.com/Gravity-Tech/gateway-deployer/ethereum/deployer"
	"github.com/ethereum/go-ethereum/common"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/urfave/cli/v2"
)

type Network string
type ChainId string

const (
	DefaultConfig = "ethereum-cfg.json"
)

const (
	EvmBasedDirection = "evm-based"
	NonEvmBasedDirection = "non-evm-based"
)

var (
	EthereumCommand = &cli.Command{
		Name:        "ethereum",
		Usage:       "",
		Description: "",
		Subcommands: []*cli.Command{
			{
				Name:        "deploy",
				Usage:       "Deploy contracts",
				Description: "",
				Action:      deploy,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  ConfigFlag,
						Value: DefaultConfig,
					},
					&cli.StringFlag{
						Name:  "direction",
						Value: NonEvmBasedDirection,
					},
				},
			},
		},
	}
)

func deploy(ctx *cli.Context) error {
	cfgPath := ctx.String(ConfigFlag)
	cfgDirection := ctx.String("direction")

	cfg := new(config.EthereumConfig)
	config.ParseConfig(cfgPath, cfg)
	err := cfg.Validate()
	if err != nil {
		return err
	}

	fmt.Println("Deploy ethereum contracts")
	fmt.Printf("Node url: %s\n", cfg.NodeUrl)

	ethClient, err := ethclient.DialContext(ctx.Context, cfg.NodeUrl)
	if err != nil {
		return err
	}

	privKey := os.Getenv("DEPLOYER_PRIV_KEY")

	privateKey, err := crypto.HexToECDSA(privKey)
	if err != nil {
		return err
	}

	transactor := bind.NewKeyedTransactor(privateKey)
	ethDeployer := deployer.NewEthDeployer(ethClient, transactor)

	fmt.Println("Deploy gravity contract")

	gravityAddress := cfg.ExistingGravityAddress

	fmt.Printf("Gravity address: %s\n", gravityAddress)

	var portType deployer.PortType
	switch cfgDirection {
	case NonEvmBasedDirection:
		portType = deployer.IBPort
	case EvmBasedDirection:
		portType = deployer.LUPort
	}

	var consulsList []common.Address
	for _, consul := range cfg.ConsulsAddress {
		consulsList = append(consulsList, common.HexToAddress(consul))
	}

	port, err := ethDeployer.DeployPort(
		gravityAddress,
		int(deployer.BytesType),
		cfg.ExistingTokenAddress,
		consulsList,
		cfg.GravityBftCoefficient,
		portType,
		ctx.Context,
	)
	if err != nil {
		return err
	}

	fmt.Printf("Gravity address: %s\n", gravityAddress)

	fmt.Printf("---------%v---------: \n", portType.Format())
	fmt.Printf("Port address: %s\n", port.PortAddress)
	fmt.Printf("Nebula address: %s\n", port.NebulaAddress)
	fmt.Printf("Token address: %s\n", port.ERC20Address)

	return nil
}
func faucet(ctx *cli.Context) error {
	args := ctx.Args()

	cfgPath := ctx.String(ConfigFlag)
	cfg := new(config.EthereumConfig)
	config.ParseConfig(cfgPath, cfg)

	erc20Address := args.Get(0)
	receiver := args.Get(1)
	amount, err := strconv.ParseInt(args.Get(2), 10, 64)
	if err != nil {
		return err
	}

	fmt.Printf("ERC20 Address: %s\n", erc20Address)
	fmt.Printf("Receiver: %s\n", receiver)
	fmt.Printf("Amount: %d\n", amount)

	ethClient, err := ethclient.DialContext(ctx.Context, cfg.NodeUrl)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(cfg.PrivKey)
	if err != nil {
		return err
	}

	ethDeployer := deployer.NewEthDeployer(ethClient, bind.NewKeyedTransactor(privateKey))

	hash, err := ethDeployer.Faucet(erc20Address, receiver, amount, ctx.Context)
	if err != nil {
		return err
	}

	fmt.Printf("Tx id: %s\n", hash)

	return nil
}
