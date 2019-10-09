package repo

import (
	"context"
	"grpc-account-svc/api"
)

type Account struct {
	AccountId string
}

func GetAccountById(ctx context.Context, id string) (*api.Account, error) {

	return newAccount(), nil
}

func GetTopAccounts(ctx context.Context, count int64) ([]*api.Account, error) {

	return []*api.Account{newAccount(), newAccount(), newAccount()}, nil
}

func newAccount() *api.Account {
	return &api.Account{
		AccountId: "123",
	}
}
