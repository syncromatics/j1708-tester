package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/syncromatics/j1708-tester/internal/web"
	"github.com/syncromatics/j1708-tester/pkg/common"
	"github.com/syncromatics/j1708-tester/pkg/simma"

	"golang.org/x/sync/errgroup"
)

var interpreter = &common.J1587Interpreter{}

var addr = flag.String("addr", ":8080", "http service address")

var hub *web.Hub

func main() {
	d := simma.NewDevice("/dev/serial/by-id/usb-Simma_Software_VNA2-USB_1-if00", printMessages)

	proxy := common.NewSendProxy(d)

	hub = web.NewHub(proxy.Send)
	go hub.Run()

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		web.ServeWs(hub, w, r)
	})

	ctx, cancel := context.WithCancel(context.Background())
	grp, ctx := errgroup.WithContext(ctx)

	grp.Go(d.Open(ctx))
	grp.Go(hostWeb())

	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-waiter:
	case <-ctx.Done():
	}
	cancel()
	if err := grp.Wait(); err != nil {
		panic(err)
	}
}

func hostWeb() func() error {
	return func() error {
		err := http.ListenAndServe(*addr, nil)
		if err != nil {
			return err
		}
		return nil
	}
}

func printMessages(m *common.J1587Message) {
	s, err := interpreter.Interpret(m)
	if err != nil {
		return
	}
	hub.Broadcast(s)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}
