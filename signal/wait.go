package signal

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func WaitForTerminationSignal() {

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Shutting down...")

	signal.Stop(sigCh)
}
