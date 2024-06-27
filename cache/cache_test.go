package cache_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isp-gate-service/cache"
)

func TestGetBasic(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	cache := cache.New()
	cache.Set("key", []byte("data"), 24*time.Hour)

	data, ok := cache.Get("key")
	require.True(ok)
	require.EqualValues("data", data)

	_, ok = cache.Get("key2")
	require.False(ok)
}

func TestGetExpired(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	cache := cache.New()
	cache.Set("key", []byte("data"), 500*time.Millisecond)

	time.Sleep(1 * time.Second)

	data, ok := cache.Get("key")
	require.False(ok)
	require.Nil(data)
}

func TestNilValue(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	cache := cache.New()
	cache.Set("key", nil, 24*time.Hour)

	data, ok := cache.Get("key")
	require.True(ok)
	require.Nil(data)
}
