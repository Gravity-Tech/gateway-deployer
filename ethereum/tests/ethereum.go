package tests

import (
	"testing"
	"encoding/hex"
    "os"
    "log"
	"math/big"
    "context"
    "crypto/ecdsa"
	"math/rand"
	"rh_tests/helpers"
	"github.com/Gravity-Tech/gravity-core/common/contracts"
	"rh_tests/api/ibport"
	"rh_tests/api/luport"
	"github.com/ethereum/go-ethereum/crypto"
    "github.com/ethereum/go-ethereum/accounts/abi/bind"
    "github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var ethConnection *ethclient.Client
var config helpers.Config
var addresses helpers.DeployedAddresses
var nebulaContract *contracts.Nebula
var nebulaReverseContract *contracts.Nebula
var ibportContract *ibport.IBPort
var luportContract *luport.LUPort

type Request struct {
	Id *big.Int
	HomeAddress common.Address
	ForeightAddress [32]byte
	Amount *big.Int
	Status uint8
}

func ReadConfig() {
	var err error
	config, err = helpers.LoadConfiguration()
    if err != nil {
        log.Fatal(err)
    }
	addresses, err = helpers.LoadAddresses()
    if err != nil {
        log.Fatal(err)
    }
}

func ConnectClient() bool {
    conn, err := ethclient.Dial(config.Endpoint)
    if err != nil {
        log.Fatal(err)
        return false
    }
    ethConnection = conn
    return true
}

func BindContracts() {
	var err error
	nebulaContract, err = contracts.NewNebula(common.HexToAddress(addresses.Nebula), ethConnection)
    if err != nil {
        log.Fatal(err)
    }
	nebulaReverseContract, err = contracts.NewNebula(common.HexToAddress(addresses.NebulaReverse), ethConnection)
    if err != nil {
        log.Fatal(err)
    }
	ibportContract, err = ibport.NewIBPort(common.HexToAddress(addresses.IBPort), ethConnection)
    if err != nil {
        log.Fatal(err)
    }
	luportContract, err = luport.NewLUPort(common.HexToAddress(addresses.LUPort), ethConnection)
    if err != nil {
        log.Fatal(err)
    }
}

func signData(dataHash [32]byte, validSignsCount int, isReverse bool) (*big.Int) {
    var r [5][32]byte
    var s [5][32]byte
	var v [5]uint8
	
    privateKey, err := crypto.HexToECDSA(config.OraclePK[0])
    if err != nil {
        log.Fatal(err)
    }

    publicKey := privateKey.Public()
    publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
    if !ok {
        log.Fatal("error casting public key to ECDSA")
    }

    fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
    nonce, err := ethConnection.PendingNonceAt(context.Background(), fromAddress)
    if err != nil {
        log.Fatal(err)
    }

    gasPrice, err := ethConnection.SuggestGasPrice(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    auth := bind.NewKeyedTransactor(privateKey)
    auth.Nonce = big.NewInt(int64(nonce))
    auth.Value = big.NewInt(0)     // in wei
    auth.GasLimit = uint64(300000) // in units
    auth.GasPrice = gasPrice

    for position, validatorKey := range config.OraclePK {
        validatorEthKey, _ := crypto.HexToECDSA(validatorKey)
        sign, _ := crypto.Sign(dataHash[:], validatorEthKey)

        copy(r[position][:], sign[:32])
        copy(s[position][:], sign[32:64])

        if (position < validSignsCount) {
            v[position] = sign[64] + 27
        } else {
            v[position] = 0 // generate invalid signature
        }
	}
	
	var tx *types.Transaction

	if !isReverse {
    	tx, err = nebulaContract.SendHashValue(auth, dataHash, v[:], r[:], s[:])
		if err != nil {
			return nil
		}
	} else {
    	tx, err = nebulaReverseContract.SendHashValue(auth, dataHash, v[:], r[:], s[:])
		if err != nil {
			return nil
		}
	}
	
	receipt, err := bind.WaitMined(context.Background(), ethConnection, tx)
    if err != nil {
        log.Fatal(err)
	}

	if len(receipt.Logs) < 1 {
		return nil
	}

	if !isReverse {
		pulseEvent, err := nebulaContract.NebulaFilterer.ParseNewPulse(*receipt.Logs[0])
		if err != nil {
			log.Fatal(err)
		}
		return pulseEvent.PulseId
	} else {
		pulseEvent, err := nebulaReverseContract.NebulaFilterer.ParseNewPulse(*receipt.Logs[0])
		if err != nil {
			log.Fatal(err)
		}
		return pulseEvent.PulseId
	}
}

func TestMain(m *testing.M) {
    ReadConfig()
	ConnectClient()
	BindContracts()
	os.Exit(m.Run())
}

func Random32Byte() [32]byte {
	var out [32]byte
	rand.Read(out[:])
	return out
}

func filluint(data []byte, pos uint, val uint64) {
	var i int
	for i = 31; i >= 0; i-- {
		data[i + int(pos)] = byte(val % 256)
		val = val / 256
	}
}

func filladdress(data []byte, pos uint, addressStr string) {
	address := common.HexToAddress(addressStr)
	copy(data[pos:], address[:])
}

func bytes32fromhex(s string) ([32]byte) {
	var ret [32]byte
	decoded, err := hex.DecodeString(s[:])
	if err != nil {
		return ret
	}
	copy(ret[:], decoded[:])
	return ret
}

func sendData(key string, value []byte, blockNumber *big.Int, subscriptionId [32]byte, isReverse bool) (bool) {
    privateKey, err := crypto.HexToECDSA(key)
    if err != nil {
        log.Fatal(err)
    }
	auth := bind.NewKeyedTransactor(privateKey)
	if !isReverse {
		tx, err := nebulaContract.SendValueToSubByte(auth, value, blockNumber, subscriptionId)
		if err != nil {
			return false
		}
			bind.WaitMined(context.Background(), ethConnection, tx)
		} else {
			tx, err := nebulaReverseContract.SendValueToSubByte(auth, value, blockNumber, subscriptionId)
			if err != nil {
				log.Fatal(err)
				return false
			}
			bind.WaitMined(context.Background(), ethConnection, tx)
		}
    return true
}


func TestPulseSaved(t *testing.T) {
	d := Random32Byte()
	pulseId := signData(d, 5, false)
	if pulseId == nil {
		t.Error("can't send signed data")
	} else {
		pulseData, err := nebulaContract.Pulses(nil, pulseId)

		if err != nil {
			t.Error("can't get pulse hash")
		}

		if d != pulseData.DataHash {
			t.Error("data mismatch")
		}
	}
}

func TestPulseCorrect3(t *testing.T) {
	d := Random32Byte()
	if signData(d, 3, false) == nil {
		t.Error("can't send signed data")
	}
}

func TestPulseInCorrect2(t *testing.T) {
	d := Random32Byte()
	if signData(d, 2, false) != nil {
		t.Error("transaction 2/3 valid sigs should be rejected")
	}
}

func TestInvalidHash(t *testing.T) {
	var attachedData [2]byte
	attachedData[0] = 1
	attachedData[1] = 2

	// generate invalid proof
	proof := crypto.Keccak256Hash(attachedData[:])

	pulseId := signData(proof, 5, false)
	if pulseId == nil {
		t.Error("can't submit proof")
	} else if sendData(config.OraclePK[0], attachedData[:], pulseId, bytes32fromhex(addresses.SubscriptionId), false) {
		t.Error("this tx should fail because of invalid hash")
	}
}

func TestMint(t *testing.T) {
	var attachedData [1+32+32+20]byte

	attachedData[0] = 'm' // mint
	filluint(attachedData[:], 1, 123456789) // req id
	filluint(attachedData[:], 1+32, 10) // amount = 10
	filladdress(attachedData[:], 1+32+32, "9561C133DD8580860B6b7E504bC5Aa500f0f0103") // address

	proof := crypto.Keccak256Hash(attachedData[:])
	pulseId := signData(proof, 5, false)
	if pulseId == nil {
		t.Error("can't submit proof")
	} else if !sendData(config.OraclePK[0], attachedData[:], pulseId, bytes32fromhex(addresses.SubscriptionId), false) {
		t.Error("can't submit data")
	}
}

func TestChangeStatusFail(t *testing.T) {
	var attachedData [1+32+1]byte

	attachedData[0] = 'c' // change status
	filluint(attachedData[:], 1, 111111111) // req id = 111111111
	attachedData[1+32] = 1

	proof := crypto.Keccak256Hash(attachedData[:])
	pulseId := signData(proof, 5, false)
	if pulseId == nil {
		t.Error("can't submit proof")
	}

	if sendData(config.OraclePK[0], attachedData[:], pulseId, bytes32fromhex(addresses.SubscriptionId), false) {
		t.Error("request should fail")
	}
}

func TestChangeStatusOk(t *testing.T) {
	var dummyAddress [32]byte

	privateKey, err := crypto.HexToECDSA(config.OraclePK[0])
	if err != nil {
		t.Error(err)
	}

	auth := bind.NewKeyedTransactor(privateKey)

	tx, err := ibportContract.CreateTransferUnwrapRequest(auth, big.NewInt(10000000), dummyAddress)
    if err != nil {
        log.Fatal(err)
	}

	receipt, err := bind.WaitMined(context.Background(), ethConnection, tx)

	if len(receipt.Logs) < 3 {
		log.Fatal("no log entry")
	}
	
	requestCreatedEvent, err := ibportContract.IBPortFilterer.ParseRequestCreated(*receipt.Logs[2])
    if err != nil {
        log.Fatal(err)
	}

	reqId := requestCreatedEvent.Arg0

	if reqId == nil {
		t.Error("request failed")
	} else {
		var attachedData [1+32+1]byte
		attachedData[0] = 'c' // change status
		filluint(attachedData[:], 1, reqId.Uint64())

		attachedData[1+32] = 2 // next status

		proof := crypto.Keccak256Hash(attachedData[:])
		pulseId := signData(proof, 5, false)
		if pulseId == nil {
			t.Error("can't submit proof")
		}

		if !sendData(config.OraclePK[0], attachedData[:], pulseId, bytes32fromhex(addresses.SubscriptionId), false) {
			t.Error("request failed", pulseId)
		}
	}
}

func TestLock(t *testing.T) {
	var dummyAddress [32]byte

	amount := big.NewInt(12345)

	privateKey, err := crypto.HexToECDSA(config.OraclePK[0])
	if err != nil {
		t.Error(err)
	}

	auth := bind.NewKeyedTransactor(privateKey)

	tx, err := luportContract.CreateTransferUnwrapRequest(auth, amount, dummyAddress)
	if err != nil {
		t.Error(err)
	} else {
		receipt, err := bind.WaitMined(context.Background(), ethConnection, tx)
		newRequestEvent, err := luportContract.LUPortFilterer.ParseNewRequest(*receipt.Logs[2])
		if err != nil {
			t.Error(err)
		} else {
			requestStruct, _ := luportContract.Requests(nil, newRequestEvent.SwapId);
			if requestStruct.Amount.Cmp(amount) != 0 {
				t.Error("failed")
			} else if requestStruct.Status != 1 {
				t.Error("unexpected status", requestStruct.Status)
			}
		}
	}
}

func TestUnlock(t *testing.T) {
	var attachedData [1+32+32+20]byte

	attachedData[0] = 'u' // unlock
	filluint(attachedData[:], 1, 123456789) // req id
	filluint(attachedData[:], 1+32, 10) // amount = 10
	filladdress(attachedData[:], 1+32+32, "9561C133DD8580860B6b7E504bC5Aa500f0f0103") // address

	proof := crypto.Keccak256Hash(attachedData[:])
	pulseId := signData(proof, 5, true)
	if pulseId == nil {
		t.Error("can't submit proof")
	} else if !sendData(config.OraclePK[0], attachedData[:], pulseId, bytes32fromhex(addresses.ReverseSubscriptionId), true) {
		t.Error("can't submit data")
	}
}

func getRequestsQueue() ([]Request) {
	id, homeAddress, foreightAddress, amount, status, err := luportContract.GetRequests(nil)

	if err != nil {
		log.Fatal(err)
	}
	length := len(id)

	if (length != len(homeAddress)) || (length != len(foreightAddress)) || (length != len(amount)) || (length != len(status)) {
		log.Fatal("invalid response")
	}

	ret := make([]Request, length)

	for i := 0; i < length; i++ {
		ret[i] = Request{Id: id[i], HomeAddress: homeAddress[i], ForeightAddress: foreightAddress[i], Amount: amount[i], Status: status[i]}
	}

	return ret
}

func findRequestById(requests []Request, id *big.Int) (*Request) {
	for i := 0; i < len(requests); i++ {
		r := requests[i]
		if r.Id.Cmp(id) == 0 {
			return &r
		}
	}
	return nil
}

func TestApprove(t *testing.T) {
	var attachedData [1+32]byte
	var dummyAddress [32]byte

	amount := big.NewInt(12345)

	privateKey, err := crypto.HexToECDSA(config.OraclePK[0])
	if err != nil {
		t.Error(err)
	}

	auth := bind.NewKeyedTransactor(privateKey)

	tx, err := luportContract.CreateTransferUnwrapRequest(auth, amount, dummyAddress)
	if err != nil {
		t.Error(err)
	} else {
		receipt, err := bind.WaitMined(context.Background(), ethConnection, tx)
		newRequestEvent, err := luportContract.LUPortFilterer.ParseNewRequest(*receipt.Logs[2])
		if err != nil {
			t.Error(err)
		} else {
			requests := getRequestsQueue()
			r := findRequestById(requests, newRequestEvent.SwapId)
			if r == nil {
				t.Error("no request in queue")
			}
			attachedData[0] = 'a' // approve
			filluint(attachedData[:], 1, newRequestEvent.SwapId.Uint64()) // req id
			proof := crypto.Keccak256Hash(attachedData[:])
			pulseId := signData(proof, 5, true)
			if pulseId == nil {
				t.Error("can't submit proof")
			} else if !sendData(config.OraclePK[0], attachedData[:], pulseId, bytes32fromhex(addresses.ReverseSubscriptionId), true) {
				t.Error("can't submit data")
			}
			requests = getRequestsQueue()
			r = findRequestById(requests, newRequestEvent.SwapId)
			if r != nil {
				t.Error("request should not be in the queue")
			}
		}
	}
}
