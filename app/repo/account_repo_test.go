package repo

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
)

func Test_GetAccountById(t *testing.T) {
	ctx := context.Background()
	acc, err := GetAccountById(ctx, "5010060647")
	assert.Nil(t, err)
	assert.Equal(t, "5010060647", acc.AccountId)

	t1, err := ptypes.Timestamp(acc.StatusLastUpdated)
	assert.NotNil(t, t1)
	assert.Nil(t, err)
}

func Test_GetTopAccounts(t *testing.T) {
	ctx := context.Background()
	accs, err := GetTopAccounts(ctx, 10)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(accs))

}
