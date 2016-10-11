package edit

import (
	"os"
	"os/signal"
	"syscall"
)

func initCatchSignals(f func(os.Signal)) {
	ch := make(chan os.Signal)
	go func() {
		for {
			sig := <-ch
			f(sig)
		}
	}()
	signal.Notify(ch,
		os.Interrupt, // syscall.SIGINT, ctrl+c
		os.Kill,      // syscall.SIGKILL
		syscall.SIGHUP,
		syscall.SIGTERM,
		syscall.SIGQUIT)
}
