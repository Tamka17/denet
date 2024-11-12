package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log"

	pb "denet/internal/proto"

	"github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("127.0.0.1:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAccountServiceClient(conn)

	privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate key: %v", err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	// сообщение для подписи
	message := []byte("some message")

	// hash сообщения
	hash := sha256.Sum256(message)

	// подпись hash
	signature, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		log.Fatalf("failed to sign message: %v", err)
	}

	// кодировка подписи в Bace64
	signatureBase64 := base64.StdEncoding.EncodeToString(signature)

	resp, err := client.GetAccount(context.Background(), &pb.GetAccountRequest{
		EthereumAddress: address,
		CryptoSignature: signatureBase64, // We transmit the encrypted signature
	})
	if err != nil {
		log.Fatalf("error calling GetAccount: %v", err)
	}

	log.Printf("Gastoken Balance: %s, Wallet Nonce: %d", resp.GetGastokenBalance(), resp.GetWalletNonce())
}
