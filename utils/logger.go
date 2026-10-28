package utils

import (
	"os"

	"github.com/charmbracelet/log"
)

// Logger é o registrador compartilhado do programa, escrito em stderr.
var Logger = novoLogger()

func novoLogger() *log.Logger {
	l := log.New(os.Stderr)
	l.SetLevel(log.InfoLevel)
	l.SetReportTimestamp(true)
	l.SetTimeFormat("2006-01-02 15:04:05")
	return l
}
