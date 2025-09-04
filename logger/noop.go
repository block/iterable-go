package logger

type Noop struct {
}

var _ Logger = &Noop{}

func (n Noop) Debugf(format string, args ...any) {
}

func (n Noop) Infof(format string, args ...any) {
}

func (n Noop) Warnf(format string, args ...any) {
}

func (n Noop) Errorf(format string, args ...any) {
}
