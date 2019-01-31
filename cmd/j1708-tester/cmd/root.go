package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rakyll/statik/fs"
	"github.com/syncromatics/j1708-tester/internal/web"
	"github.com/syncromatics/j1708-tester/pkg/common"
	"github.com/syncromatics/j1708-tester/pkg/simma"

	astilectron "github.com/asticode/go-astilectron"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	device      *string
	port        *int
	hub         *web.Hub
	interpreter = &common.J1587Interpreter{}
	addr        *string
	noUI        *bool
)

var rootCmd = &cobra.Command{
	Use:   "j1708-tester",
	Short: "j1708-tester is a tool to test vehicle networks",
	Long:  "j1708-tester is a tool to test vehicle networks",
	Run: func(cmd *cobra.Command, args []string) {
		if port == nil {
			a := 8080
			port = &a
		}

		a := fmt.Sprintf(":%d", *port)
		addr = &a

		if device == nil || *device == "" {
			device = getDefaultDevice()
		}

		d := simma.NewDevice(*device, printMessages)

		proxy := common.NewSendProxy(d)

		hub = web.NewHub(proxy.Send)
		go hub.Run()

		statikFS, err := fs.New()
		if err != nil {
			log.Fatal(err)
		}

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			f, err := statikFS.Open("/index.html")
			if err != nil {
				log.Printf("get file failed: %v", err)
				return
			}
			defer f.Close()

			http.ServeContent(w, r, "index.html", time.Now(), f)
		})

		http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			web.ServeWs(hub, w, r)
		})

		ctx, cancel := context.WithCancel(context.Background())
		grp, ctx := errgroup.WithContext(ctx)

		grp.Go(d.Open(ctx))
		grp.Go(hostWeb(ctx))

		log.Printf("hosting web at http://localhost:%d...\n", *port)
		log.Println("")
		log.Println("press CTRL+C to exit.")

		if noUI == nil || !*noUI {
			grp.Go(startUI(cancel))
		}

		waiter := make(chan os.Signal, 1)
		signal.Notify(waiter, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-waiter:
		case <-ctx.Done():
		}

		log.Println("exiting...")

		cancel()
		if err := grp.Wait(); err != nil {
			panic(err)
		}
	},
}

func init() {
	port = rootCmd.Flags().IntP("port", "p", 8080, "The port to host the server on")
	device = rootCmd.Flags().StringP("device", "d", "", "The vehicle network device")
	noUI = rootCmd.Flags().Bool("no-ui", false, "setting this flag disables the ui")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func hostWeb(ctx context.Context) func() error {
	srv := &http.Server{Addr: *addr}

	cancel := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			srv.Shutdown(context.Background())
			return
		case <-cancel:
			return
		}
	}()

	return func() error {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
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

func getDefaultDevice() *string {
	d := ""
	switch runtime.GOOS {
	case "windows":
		d = "COM1"
		break
	case "linux":
		d = "/dev/serial/by-id/usb-Simma_Software_VNA2-USB_1-if00"
		break
	case "darwin":
		d = "/dev/serial/by-id/usb-Simma_Software_VNA2-USB_1-if00"
		break
	}
	return &d
}

func startUI(cancel func()) func() error {
	return func() error {
		home, err := homedir.Dir()
		if err != nil {
			return errors.Wrap(err, "failed getting home directory")
		}

		a, err := astilectron.New(astilectron.Options{
			AppName:           "J1708/1587 Tester",
			BaseDirectoryPath: fmt.Sprintf("%s/.j1708tester/", home),
		})
		if err != nil {
			return err
		}
		defer a.Close()

		err = a.Start()
		if err != nil {
			return err
		}

		w, err := a.NewWindow(fmt.Sprintf("http://127.0.0.1:%d", *port), &astilectron.WindowOptions{
			Center: astilectron.PtrBool(true),
			Height: astilectron.PtrInt(600),
			Width:  astilectron.PtrInt(600),
		})
		if err != nil {
			return errors.Wrap(err, "failed to new window")
		}

		err = w.Create()
		if err != nil {
			return errors.Wrap(err, "failed to create window")
		}

		w.On(astilectron.EventNameWindowEventClosed, func(e astilectron.Event) (deleteListener bool) {
			cancel()
			a.Stop()
			return
		})

		a.Wait()

		return nil
	}
}
