package deployer

import (
	"context"
	"integration-test/config"
	"math/big"

	"github.com/Gravity-Tech/gateway/abi/ethereum/erc20"

	"github.com/Gravity-Tech/gateway/abi/ethereum/ibport"
	"github.com/Gravity-Tech/gateway/abi/ethereum/luport"
	"github.com/Gravity-Tech/gravity-core/abi/ethereum"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	LUPort PortType = iota
	IBPort
)

type PortType int

type GatewayPort struct {
	PortAddress   string
	NebulaAddress string
	ERC20Address  string
}

type EthDeployer struct {
	ethClient  *ethclient.Client
	transactor *bind.TransactOpts
}

func NewEthDeployer(ethClient *ethclient.Client, transactor *bind.TransactOpts) *EthDeployer {
	return &EthDeployer{
		ethClient:  ethClient,
		transactor: transactor,
	}
}

func (deployer *EthDeployer) DeployPort(gravityAddress string, dataType int, erc20Setting config.NewERC20,
	oracles []common.Address, bftCoefficient int, portType PortType, ctx context.Context) (*GatewayPort, error) {

	erc20Address, tx, erc20Token, err := erc20.DeployToken(
		deployer.transactor,
		deployer.ethClient,
		erc20Setting.Name,
		erc20Setting.Symbol,
	)
	if err != nil {
		return nil, err
	}

	_, err = bind.WaitMined(ctx, deployer.ethClient, tx)
	if err != nil {
		return nil, err
	}

	nebulaAddress, tx, nebula, err := ethereum.DeployNebula(
		deployer.transactor,
		deployer.ethClient,
		uint8(dataType),
		common.HexToAddress(gravityAddress),
		oracles,
		big.NewInt(int64(bftCoefficient)),
	)
	if err != nil {
		return nil, err
	}

	_, err = bind.WaitMined(ctx, deployer.ethClient, tx)
	if err != nil {
		return nil, err
	}

	var portAddress common.Address
	switch portType {
	case IBPort:
		portAddress, tx, _, err = ibport.DeployIBPort(deployer.transactor, deployer.ethClient, nebulaAddress, erc20Address)
	case LUPort:
		portAddress, tx, _, err = luport.DeployLUPort(deployer.transactor, deployer.ethClient, nebulaAddress, erc20Address)
	}
	if err != nil {
		return nil, err
	}

	_, err = bind.WaitMined(ctx, deployer.ethClient, tx)
	if err != nil {
		return nil, err
	}

	var owner common.Address

	switch portType {
	case IBPort:
		owner = portAddress
	case LUPort:
		owner = common.HexToAddress(erc20Setting.TokenOwnerForLUPort)
	}

	tx, err = erc20Token.AddMinter(deployer.transactor, owner)
	if err != nil {
		return nil, err
	}

	_, err = bind.WaitMined(ctx, deployer.ethClient, tx)
	if err != nil {
		return nil, err
	}

	tx, err = nebula.Subscribe(deployer.transactor, portAddress, 1, big.NewInt(0))
	if err != nil {
		return nil, err
	}

	_, err = bind.WaitMined(ctx, deployer.ethClient, tx)
	if err != nil {
		return nil, err
	}

	return &GatewayPort{
		PortAddress:   portAddress.Hex(),
		NebulaAddress: nebulaAddress.Hex(),
		ERC20Address:  erc20Address.Hex(),
	}, nil
}

func (deployer *EthDeployer) Fauset(erc20Address string, receiver string, amount int64, ctx context.Context) (string, error) {
	erc20Token, err := erc20.NewToken(common.HexToAddress(erc20Address), deployer.ethClient)
	if err != nil {
		return "", err
	}

	decimals, err := erc20Token.Decimals(nil)
	if err != nil {
		return "", err
	}

	value := big.NewInt(int64(amount))
	value.Mul(value, big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	tx, err := erc20Token.Mint(deployer.transactor, common.HexToAddress(receiver), value)
	if err != nil {
		return "", err
	}

	_, err = bind.WaitMined(ctx, deployer.ethClient, tx)
	if err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}
func (deployer *EthDeployer) DeployGravity(consuls [5]string, bftCoefficient int, ctx context.Context) (string, error) {
	var consulsAddress []common.Address

	for _, v := range consuls {
		consulsAddress = append(consulsAddress, common.HexToAddress(v))
	}

	gravityAddress, tx, _, err := ethereum.DeployGravity(deployer.transactor, deployer.ethClient, consulsAddress[:], big.NewInt(int64(bftCoefficient)))
	if err != nil {
		return "", err
	}

	_, err = bind.WaitMined(ctx, deployer.ethClient, tx)
	if err != nil {
		return "", err
	}

	return gravityAddress.Hex(), nil
}
