package logger

import (
	"fmt"
)

type stdOut struct {
	print func(msg string)
}

var _ Logger = &stdOut{}

func NewStdOut() Logger {
	return &stdOut{
		print: func(msg string) {
			fmt.Println(msg)
		},
	}
}

func (p *stdOut) Debugf(format string, args ...any) {
	p.print(fmt.Sprintf("[DEBUG] "+format, args...))
}

func (p *stdOut) Infof(format string, args ...any) {
	p.print(fmt.Sprintf("[INFO] "+format, args...))
}

func (p *stdOut) Warnf(format string, args ...any) {
	p.print(fmt.Sprintf("[WARN] "+format, args...))
}

func (p *stdOut) Errorf(format string, args ...any) {
	p.print(fmt.Sprintf("[ERROR] "+format, args...))
}
