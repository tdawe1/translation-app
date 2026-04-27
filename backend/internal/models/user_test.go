package models

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

func TestWatcherStateUserIDIsPrimaryKey(t *testing.T) {
	parsed, err := schema.Parse(&WatcherState{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)

	field := parsed.LookUpField("UserID")
	require.NotNil(t, field)
	require.True(t, field.PrimaryKey)
}
