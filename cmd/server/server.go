package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"net"

	pb "denet/internal/proto"
	erc20 "denet/pkg/erc20"
	"io"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
)

// рефлизация методов gRPC
type Server struct {
	pb.UnimplementedAccountServiceServer
	client *ethclient.Client
}

// GenerateECDSAKeys генерация секртеных и открытых пар ключей
func GenerateECDSAKeys() (*ecdsa.PrivateKey, error) {
	// приватный ключ
	privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// SignMessage подпись, использую закрытый ключ
func SignMessage(privateKey *ecdsa.PrivateKey, msg []byte) ([]byte, error) {
	hash := sha256.Sum256(msg)
	signature, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

// VerifySignature проверка подписи
func VerifySignature(pubKey *ecdsa.PublicKey, msg []byte, signature []byte) bool {
	hash := sha256.Sum256(msg)
	recoveredPubKey, err := crypto.SigToPub(hash[:], signature)
	if err != nil {
		return false
	}
	return recoveredPubKey.Equal(pubKey)
}

// GetAccount method
func (s *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {

	if s.client == nil {
		return nil, fmt.Errorf("Ethereum client is not initialized")
	}

	ethereumAddress := common.HexToAddress(req.GetEthereumAddress())
	cryptoSignature := req.GetCryptoSignature()

	signature, err := base64.StdEncoding.DecodeString(cryptoSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %v", err)
	}

	// проверка подписи
	if !isValidSignature(ethereumAddress, signature) {
		return nil, fmt.Errorf("invalid signature")
	}

	tokenAddress := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")

	gasTokenBalance, err := GetTokenBalance(s.client, tokenAddress, ethereumAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get token balance: %v", err)
	}

	walletNonce, err := GetNonce(s.client, ethereumAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %v", err)
	}

	return &pb.GetAccountResponse{
		GastokenBalance: gasTokenBalance.String(),
		WalletNonce:     walletNonce,
	}, nil
}

// GetTokenBalance  извлекает баланс токена ERC-20 для данного адреса
func GetTokenBalance(client *ethclient.Client, tokenAddress common.Address, owner common.Address) (*big.Int, error) {
	// Получение контракта ERC-20
	contract, err := erc20.NewErc20(tokenAddress, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract instance: %v", err)
	}

	// Получение баланса
	balance, err := contract.BalanceOf(&bind.CallOpts{}, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve balance: %v", err)
	}

	return balance, nil
}

// GetNonce извлекает одноразовый номер
func GetNonce(client *ethclient.Client, address common.Address) (uint64, error) {
	nonce, err := client.PendingNonceAt(context.Background(), address)
	if err != nil {
		return 0, fmt.Errorf("failed to get nonce: %v", err)
	}
	return nonce, nil
}

// GetAccounts method (поток, stream)
func (s *Server) GetAccounts(stream pb.AccountService_GetAccountsServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Получение баланса для каждого адреса Ethereum в запросе
		for _, ethAddress := range req.GetEthereumAddresses() {
			balance := getERC20Balance(s.client, ethAddress, req.GetErc20TokenAddress())

			// Stream the response back to the client
			response := &pb.GetAccountsResponse{
				EthereumAddress: ethAddress,
				Erc20Balance:    balance,
			}
			if err := stream.Send(response); err != nil {
				return err
			}
		}
	}
	return nil
}

// getERC20Balance извлекает баланс
func getERC20Balance(client *ethclient.Client, address, tokenAddress string) string {
	tokenAddressCommon := common.HexToAddress(tokenAddress)
	userAddressCommon := common.HexToAddress(address)

	// Получение копии контракта ERC-20
	contract, err := erc20.NewErc20(tokenAddressCommon, client)
	if err != nil {
		log.Fatalf("Failed to load ERC20 contract: %v", err)
	}

	// Вызов функции "balanceOf" контракта ERC-20
	balance, err := contract.BalanceOf(&bind.CallOpts{}, userAddressCommon)
	if err != nil {
		log.Fatalf("Failed to retrieve token balance: %v", err)
	}

	return balance.String()
}

// Вспомогательная функция для проверки подписи (обновлена)
func isValidSignature(address common.Address, signature []byte) bool {
	msg := []byte("some message")
	hash := sha256.Sum256(msg)

	// восстановление открытого ключа из подписи
	pubKey, err := crypto.SigToPub(hash[:], signature)
	if err != nil {
		return false
	}

	return crypto.PubkeyToAddress(*pubKey) == address
}

func main() {

	var client *ethclient.Client

	client, err := ethclient.Dial("https://mainnet.infura.io/v3/0e7d2c4248f2435da085101327eaa5e3")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// генерация ключа
	privateKey, err := GenerateECDSAKeys()
	if err != nil {
		log.Fatalf("Error generating keys: %v", err)
	}

	publicKey := &privateKey.PublicKey
	msg := []byte("test message")
	signature, err := SignMessage(privateKey, msg)
	if err != nil {
		log.Fatalf("Error signing message: %v", err)
	}

	isValid := VerifySignature(publicKey, msg, signature)
	fmt.Printf("Signature valid: %v\n", isValid)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := &Server{client: client}
	grpcServer := grpc.NewServer()
	pb.RegisterAccountServiceServer(grpcServer, s)

	log.Printf("server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
