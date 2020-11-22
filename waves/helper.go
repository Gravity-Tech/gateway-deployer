package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"rh_tests/helpers"

	wavesClient "github.com/wavesplatform/gowaves/pkg/client"

	wavesCrypto "github.com/wavesplatform/go-lib-crypto"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	RideErrorPrefix = "Error while executing account-script: "
)

type TestPulseConfig struct {
	Helper helpers.ClientHelper
	Client *wavesClient.Client
	Ctx    context.Context

	Gravity *Account
	Nebula  *Account
	Sub     *Account

	Consuls []*Account
	Oracles []*Account
}

type Config struct {
	GravityScriptFile string
	NebulaScriptFile  string
	SubMockScriptFile string
	NodeUrl           string
	DistributionSeed  string
	ChainId           byte
}

type RideErr struct {
	Message string
}

type Account struct {
	Address string
	// In case of waves: Secret is private key actually
	Secret  crypto.SecretKey
	PubKey  crypto.PublicKey
}

type DeploymentConfig struct {
	TestPulseConfig

	Consuls []string
	Oracles []string
}
type DeploymentConfigFile struct {
	Config
	GravityContractSeed    string
	NebulaContractSeed     string
	// Considered as LU_port in SuSy case
	SubscriberContractSeed string

	// TEMP:
	ConsulsAddressList     []string
}


func LoadDeploymentConfig(filename string) (DeploymentConfigFile, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return DeploymentConfigFile{}, err
	}
	config := DeploymentConfigFile{}
	if err := json.Unmarshal(file, &config); err != nil {
		return DeploymentConfigFile{}, err
	}
	return config, err
}


func LoadConfig(filename string) (Config, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	config := Config{}
	if err := json.Unmarshal(file, &config); err != nil {
		return Config{}, err
	}
	return config, err
}

func ScriptFromFile(filename string) ([]byte, error) {
	scriptBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	script, err := base64.StdEncoding.DecodeString(string(scriptBytes))
	if err != nil {
		return nil, err
	}

	return script, nil
}

func GenerateAddressFromSeed(chainId byte, wordList string) (*Account, error) {
	seed := wavesCrypto.Seed(wordList)
	wCrypto := wavesCrypto.NewWavesCrypto()
	address := string(wCrypto.AddressFromSeed(seed, wavesCrypto.WavesChainID(chainId)))
	seedWaves, err := crypto.NewSecretKeyFromBase58(string(wCrypto.PrivateKey(seed)))
	if err != nil {
		return nil, err
	}
	pubKey := wCrypto.PublicKey(seed)

	return &Account{
		Address: address,
		PubKey:  crypto.PublicKey(crypto.MustDigestFromBase58(string(pubKey))),
		Secret:  seedWaves,
	}, nil
}

func GenerateAddress(chainId byte) (*Account, error) {
	wCrypto := wavesCrypto.NewWavesCrypto()
	seed := wCrypto.RandomSeed()
	address := string(wCrypto.AddressFromSeed(seed, wavesCrypto.WavesChainID(chainId)))
	seedWaves, err := crypto.NewSecretKeyFromBase58(string(wCrypto.PrivateKey(seed)))
	if err != nil {
		return nil, err
	}
	pubKey := wCrypto.PublicKey(seed)

	return &Account{
		Address: address,
		PubKey:  crypto.PublicKey(crypto.MustDigestFromBase58(string(pubKey))),
		Secret:  seedWaves,
	}, nil
}

func CheckRideError(rideErr error, msg string) error {
	body := rideErr.(*wavesClient.RequestError).Body
	var rsError RideErr
	err := json.Unmarshal([]byte(body), &rsError)
	if err != nil {
		return err
	}
	if rsError.Message != RideErrorPrefix+msg {
		return errors.New("error not found")
	}

	return nil
}
