package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelQuiet Level = iota
	LevelInfo
	LevelVerbose
	LevelDebug
)

// Logger writes human facing diagnostics. Default output is quiet on purpose.
// Operators asked for a report, not a memoir of every git invocation.
type Logger struct {
	mu     sync.Mutex
	level  Level
	out    io.Writer
	err    io.Writer
	prefix string
}

func New(level Level) *Logger {
	return &Logger{
		level:  level,
		out:    os.Stderr,
		err:    os.Stderr,
		prefix: "forkledger",
	}
}

func (l *Logger) SetOutput(out io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = out
	l.err = out
}

func (l *Logger) Level() Level {
	return l.level
}

func (l *Logger) Info(format string, args ...any) {
	l.write(LevelInfo, format, args...)
}

func (l *Logger) Verbose(format string, args ...any) {
	l.write(LevelVerbose, format, args...)
}

func (l *Logger) Debug(format string, args ...any) {
	l.write(LevelDebug, format, args...)
}

func (l *Logger) write(min Level, format string, args ...any) {
	if l == nil || l.level < min || l.level == LevelQuiet {
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = strings.TrimRight(msg, "\n")
	stamp := time.Now().UTC().Format("15:04:05")
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.out, "%s %s %s\n", stamp, l.prefix, msg)
}
