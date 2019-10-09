package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"grpc-account-svc/api"
)

const (
	address = "[::]:3101"
)

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := api.NewAccountServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 3600*time.Second)
	defer cancel()
	r, err := c.GetTopAccounts(ctx, &api.GetTopAccountRequest{Count: 10})
	//r, err := c.GetAccount(ctx, &api.GetAccountRequest{ AccountId: "58870000580"})
	if err != nil {
		log.Fatalf("could not greet: %+v", err)
	}

	for {
		acc, err := r.Recv()
		if err != nil {
			log.Printf("error during read %+v", err)
			break
		}
		log.Println(acc)
	}
	log.Printf("Greeting: %+v\n", r)
}