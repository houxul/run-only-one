package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	. "run-only-one/core"
)

func main() {
	printCancelFunc := Run("print", print)
	defer printCancelFunc()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	// 服务停止或收到信号
	select {
	case sig := <-signals:
		log.Println("receive signal", sig.String())
	}
}

func print(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Println("hello world")
			time.Sleep(2 * time.Second)
		}
	}
}
