//nolint
package state

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	a := assert.New(t)
	newLimiter := func(count int) *limiter {
		return &limiter{timeout: 5 * time.Second, pattern: "example", pointer: -1, datetime: make([]time.Time, count)}
	}

	store12 := newLimiter(12)
	store10 := newLimiter(10)
	store7 := newLimiter(7)
	store5 := newLimiter(5)

	for i := 0; i < 7; i++ {
		expectedTrue, pointer, reqTime := store10.check()
		a.True(expectedTrue)
		store10.update(pointer, reqTime)

		expectedTrue, pointer, reqTime = store7.check()
		a.True(expectedTrue)
		store7.update(pointer, reqTime)
	}

	expectedFalse, _, _ := store7.check()
	a.False(expectedFalse)

	store12.Import(store7.Export())
	store5.Import(store10.Export())

	a.Equal(store5.datetime[0], store10.datetime[store10.pointer+1])
	a.Equal(store12.datetime[0], store7.datetime[0])
	a.Equal(store12.datetime[store7.pointer], store7.datetime[store7.pointer])
	a.Equal(store12.datetime[store7.pointer+1], time.Time{})

	expectedTrue, _, _ := store12.check()
	a.True(expectedTrue)

	for i := 0; i < 3; i++ {
		expectedTrue, pointer, reqTime := store5.check()
		a.True(expectedTrue)
		store5.update(pointer, reqTime)
	}
	expectedFalse, _, _ = store5.check()
	a.False(expectedFalse)
}
