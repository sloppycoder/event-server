package server

import (
	"context"
	"google.golang.org/grpc/grpclog"
	"log"
	"strings"

	"github.com/izumin5210/grapi/pkg/grapiserver"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"grpc-account-svc/api"
	"grpc-account-svc/app/repo"
)

// AccountServiceServer is a composite interface of api.AccountServiceServer and grapiserver.Server.
type AccountServiceServer interface {
	api.AccountServiceServer
	grapiserver.Server
}

// NewAccountServiceServer creates a new AccountServiceServer instance.
func NewAccountServiceServer() AccountServiceServer {
	return &accountServiceServerImpl{}
}

type accountServiceServerImpl struct {
}

func (s *accountServiceServerImpl) GetTopAccounts(ctx context.Context, req *api.GetTopAccountRequest) (*api.GetTopAccountResponse, error) {
	accounts, err := repo.GetTopAccounts(ctx, req.Count)

	if err != nil {
		log.Printf("error from MongoDB %+v", err)
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	return &api.GetTopAccountResponse{Accounts: accounts}, nil
}

func (s *accountServiceServerImpl) GetAccount(ctx context.Context, req *api.GetAccountRequest) (*api.Account, error) {
	id := req.AccountId
	if "" == id || "0" == id {
		grpclog.Infof("invalid account id: %s", id)
		return nil, status.Error(codes.InvalidArgument, "Invalid account id")
	}

	acc, err := repo.GetAccountById(ctx, id)
	if err == nil {
		return acc, nil
	}

	if strings.Contains(err.Error(), "no documents in result") {
		grpclog.Infof("account %s not found", id)
		return nil, status.Error(codes.NotFound, "account does not exist")
	} else {
		grpclog.Infof("unable to retrieve account summary for %s %s", id, err)
		return nil, status.Error(codes.Unavailable, "Unable to retrieve account")
	}

}
