package cmd

import (
	"fmt"
	"integration-test/config"
	"integration-test/deployer"
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
				},
			},
			{
				Name:        "fauset",
				Usage:       "Get test token",
				Description: "",
				Action:      fauset,
				ArgsUsage:   "<erc20Address> <receiver> <tokenAmount>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  ConfigFlag,
						Value: DefaultConfig,
					},
				},
			},
		},
	}
)

func deploy(ctx *cli.Context) error {
	cfgPath := ctx.String(ConfigFlag)
	cfg := new(config.EthereumConfig)
	config.ParseConfig(cfgPath, cfg)

	fmt.Println("Deploy ethereum contracts")
	fmt.Printf("Node url: %s\n", cfg.NodeUrl)

	ethClient, err := ethclient.DialContext(ctx.Context, cfg.NodeUrl)
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(cfg.PrivKey)
	if err != nil {
		return err
	}

	ethDeployer := deployer.NewEthDeployer(ethClient, bind.NewKeyedTransactor(privateKey))

	fmt.Println("Deploy gravity contract")
	gravityAddress, err := ethDeployer.DeployGravity(cfg.ConsulsAddress, cfg.GravityBftCoefficient, ctx.Context)
	if err != nil {
		return err
	}

	luPort, err := ethDeployer.DeployPort(gravityAddress, int(deployer.BytesType),
		cfg.NewERC20,
		nil,
		cfg.GravityBftCoefficient,
		deployer.LUPort,
		ctx.Context,
	)
	if err != nil {
		return err
	}

	ibPort, err := ethDeployer.DeployPort(gravityAddress, int(deployer.BytesType),
		cfg.NewERC20,
		nil,
		cfg.GravityBftCoefficient,
		deployer.IBPort,
		ctx.Context,
	)
	if err != nil {
		return err
	}

	fmt.Printf("Gravity address: %s\n", gravityAddress)

	fmt.Printf("---------LU port---------: \n")
	fmt.Printf("Port address: %s\n", luPort.PortAddress)
	fmt.Printf("Nebula address: %s\n", luPort.NebulaAddress)
	fmt.Printf("Token address: %s\n", luPort.ERC20Address)

	fmt.Printf("---------IB port---------: \n")
	fmt.Printf("Port address: %s\n", ibPort.PortAddress)
	fmt.Printf("Nebula address: %s\n", ibPort.NebulaAddress)
	fmt.Printf("Token address: %s\n", ibPort.ERC20Address)

	return nil
}
func fauset(ctx *cli.Context) error {
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

	hash, err := ethDeployer.Fauset(erc20Address, receiver, amount, ctx.Context)
	if err != nil {
		return err
	}

	fmt.Printf("Tx id: %s\n", hash)

	return nil
}
