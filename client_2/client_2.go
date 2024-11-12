package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	pb "denet/internal/proto"

	"google.golang.org/grpc"
)

func main() {
	// Connecting to the gRPC server
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewAccountServiceClient(conn)

	// генерация 10000 Ethereum адресов для теста
	testAddresses := generateEthereumAddresses(10000)

	// Тестирование метода на 100, 1000 и 10000 адресах
	testGetAccountsPerformance(client, testAddresses[:100], 3)
	testGetAccountsPerformance(client, testAddresses[:1000], 4)
	testGetAccountsPerformance(client, testAddresses[:10000], 5)
}

// тест производительности
func testGetAccountsPerformance(client pb.AccountServiceClient, addresses []string, tokenCount int) {
	start := time.Now()

	// Получение потока для отправки и получения данных
	stream, err := client.GetAccounts(context.Background())
	if err != nil {
		log.Fatalf("error opening stream: %v", err)
	}

	// Генерация реальных адресов токенов ERC-20
	tokenAddresses := generateRealTokenAddresses(tokenCount)

	// Отправка запросов на каждый токен и все адреса
	for _, tokenAddress := range tokenAddresses {
		err = stream.Send(&pb.GetAccountsRequest{
			EthereumAddresses: addresses,
			Erc20TokenAddress: tokenAddress, // по одному за раз
		})
		if err != nil {
			log.Fatalf("failed to send request: %v", err)
		}
	}

	// завершение стрима
	if err := stream.CloseSend(); err != nil {
		log.Fatalf("failed to close stream: %v", err)
	}

	// ответы от сервера
	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Printf("Stream finished: %v", err)
			break
		}
		log.Printf("Ethereum Address: %s, ERC20 Balance: %s", resp.GetEthereumAddress(), resp.GetErc20Balance())
	}

	elapsed := time.Since(start)
	log.Printf("Time elapsed for %d addresses and %d tokens: %s", len(addresses), tokenCount, elapsed)
}

// Генерация тестовых адресов Ethereum
func generateEthereumAddresses(count int) []string {
	addresses := make([]string, count)
	for i := 0; i < count; i++ {
		addresses[i] = randomHex(40) // (Ethereum address)
	}
	return addresses
}

// Мы заменяем генерацию случайных адресов токенов реальными адресами токенов ERC-20
func generateRealTokenAddresses(count int) []string {
	// Реальных адреса популярных токенов
	realTokenAddresses := []string{
		"0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
		"0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f", // UNI
		"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC
		"0x514910771af9ca656af840dff83e8264ecf986ca", // LINK
		"0x95aD61b0a150d79219B9c481f44e9943f8e8887b", // SHIB
	}

	if count > len(realTokenAddresses) {
		count = len(realTokenAddresses)
	}

	return realTokenAddresses[:count]
}

// Генерация случайной шестнадцатеричной строки
func randomHex(n int) string {
	const letters = "0123456789abcdef"
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[rand.Intn(len(letters))]
	}
	return "0x" + string(result)
}
