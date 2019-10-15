package accounts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetTopAccounts(t *testing.T) {
	ctx := context.Background()
	accs, err := GetTopAccounts(ctx, 10)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(accs))

}
