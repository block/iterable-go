package logger

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Noop(t *testing.T) {
	l := &Noop{}

	l.Debugf("debug")
	l.Infof("info")
	l.Warnf("warn")
	l.Infof("info")
}

func Test_StdOut(t *testing.T) {
	var result []string
	l := &stdOut{func(msg string) {
		result = append(result, msg)
	}}

	x := struct {
		testField string
	}{"test-field"}
	err := io.ErrClosedPipe

	l.Debugf("%s, %d, %v, %v", "Hello World!", 10, x, err)
	l.Infof("%s, %d, %v, %v", "Привет Мир!", 20, x, err)
	l.Warnf("%s, %d, %v, %v", "こんにちは世界!", 30, x, err)
	l.Errorf("%s, %d, %+v, %v", "¡Hola Mundo!", 40, x, err)
	l.Errorf("empty args")
	l.Errorf("nil args: %s", nil)
	l.Errorf("more args: %s, %s", "one")
	l.Errorf("less args: %s", "one", "two")

	assert.Equal(t, 8, len(result))
	assert.Equal(t, "[DEBUG] Hello World!, 10, {test-field}, io: read/write on closed pipe", result[0])
	assert.Equal(t, "[INFO] Привет Мир!, 20, {test-field}, io: read/write on closed pipe", result[1])
	assert.Equal(t, "[WARN] こんにちは世界!, 30, {test-field}, io: read/write on closed pipe", result[2])
	assert.Equal(t, "[ERROR] ¡Hola Mundo!, 40, {testField:test-field}, io: read/write on closed pipe", result[3])
	assert.Equal(t, "[ERROR] empty args", result[4])
	assert.Equal(t, "[ERROR] nil args: %!s(<nil>)", result[5])
	assert.Equal(t, "[ERROR] more args: one, %!s(MISSING)", result[6])
	assert.Equal(t, "[ERROR] less args: one%!(EXTRA string=two)", result[7])
}
