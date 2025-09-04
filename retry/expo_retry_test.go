package retry

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Expo_Do_count(t *testing.T) {
	err := fmt.Errorf("err")
	count := 0

	r := makeExpoRetry()
	err2 := r.Do(2, "testFnName", func(attempt int) (error, ExitStrategy) {
		assert.Equal(t, count, attempt)
		count++
		return err, Continue
	})

	assert.True(t, errors.Is(err, err2))
	assert.Equal(t, 2, count)
}

func Test_Expo_Do_returns_last_error(t *testing.T) {
	err1 := fmt.Errorf("err1")
	err2 := fmt.Errorf("err2")
	count := 0

	r := makeExpoRetry()
	err3 := r.Do(2, "testFnName", func(attempt int) (error, ExitStrategy) {
		assert.Equal(t, count, attempt)
		count++
		if count == 1 {
			return err1, Continue
		}
		return err2, Continue
	})

	assert.True(t, errors.Is(err2, err3))
	assert.Equal(t, 2, count)
}

func Test_Expo_Do_early_exit(t *testing.T) {
	err1 := fmt.Errorf("err1")
	err2 := fmt.Errorf("err2")
	count := 0

	r := makeExpoRetry()
	err3 := r.Do(10, "testFnName", func(attempt int) (error, ExitStrategy) {
		assert.Equal(t, count, attempt)
		count++
		if count == 2 {
			return err1, StopNow
		}
		return err2, Continue
	})

	assert.True(t, errors.Is(err1, err3))
	assert.Equal(t, 2, count)
}

func Test_Expo_Do_0(t *testing.T) {
	count := 0

	r := makeExpoRetry()
	err := r.Do(0, "testFnName", func(attempt int) (error, ExitStrategy) {
		count++
		assert.Fail(t, "Should never run")
		return nil, Continue
	})

	assert.Error(t, err)
	assert.Equal(t, 0, count)
}

func makeExpoRetry() *expoRetry {
	return NewExponentialRetry(
		WithInitialDuration(0 * time.Millisecond),
	).(*expoRetry)
}
