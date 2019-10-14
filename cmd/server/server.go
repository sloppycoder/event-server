package main

import (
	"context"
	"grpc-account-svc/app/server"
	"grpc-account-svc/app/sse"
	"os"

	"github.com/izumin5210/grapi/pkg/grapiserver"
	"google.golang.org/grpc/grpclog"
)

func main() {
	go sse.EventServer()

	err := run()
	if err != nil {
		grpclog.Errorf("server was shutdown with errors: %v", err)
		os.Exit(1)
	}

}

func run() error {
	ctx := context.Background()
	s := grapiserver.New(
		grapiserver.WithDefaultLogger(),
		grapiserver.WithGrpcAddr("tcp", ":3101"),
		grapiserver.WithGatewayAddr("tcp", ":3100"),
		grapiserver.WithServers(
			server.NewAccountServiceServer(),
		),
	)

	return s.Serve(ctx)
}
