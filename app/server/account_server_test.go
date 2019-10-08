package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"grpc-account-svc/api"
)

func Test_AccountServiceServer_GetAccount(t *testing.T) {
	svr := NewAccountServiceServer()
	ctx := context.Background()

	resp, err := svr.GetAccount(ctx,
		&api.GetAccountRequest{
			AccountId: "111",
		})

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.StatusLastUpdated)
	assert.EqualValues(t, len(resp.Balances), 2)
	assert.Nil(t, err)
}

func Test_AccountServiceServer_GetAccount_Invalid_AccountId(t *testing.T) {
	svr := NewAccountServiceServer()
	ctx := context.Background()

	_, err := svr.GetAccount(ctx, &api.GetAccountRequest{AccountId: ""})

	assert.NotNil(t, err)
}
