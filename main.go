package main

import (
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"net/http"
	"os"
)

type Args struct {
	Mode     string `short:"m" long:"mode" choice:"server" choice:"client" choice:"relay" default:"server" description:"Operational mode (server), (relay) or (client). default: server mode"`
	Bind     string `short:"b" long:"bind" default:":8585" description:"Binding address, where should I listen to. client default: :8585, server default: :5858"`
	Server   string `short:"s" long:"server" default:"ws://127.0.0.1:8585/ws" description:"Remote websocket server address, it should starts with ws or wss and ends with ws path ex. wss://example.com/ws"`
	Upath    string `short:"u" long:"upload" default:"127.0.0.1:5858" description:"Separate specific path for relay ex. example.com:5858"`
	Insecure bool   `short:"k" long:"insecure" default:"false" description:"Allow to connect to insecure end points default: false. (not recommended)"`
}

func main() {
	var options Args

	var parser = flags.NewParser(&options, flags.Default)
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}
	if options.Mode == "server" {
		// run server mode, ie open http server and listen to incoming requests from internet
		fmt.Println("Starting server at :3030 ...")
		srv := NewServer()
		err := srv.ListenAndServe(":3030")
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("server closed\n")
		} else if err != nil {
			fmt.Printf("error starting server: %s\n", err)
			os.Exit(1)
		}
	} else if options.Mode == "server" {
		// run client mode, ie open http server and listen to incoming requests from internet
		fmt.Println("Starting client at", options.Bind, "...")
		srv, _ := NewClient(options.Server)
		err := srv.ListenAndServe("tcp", options.Bind)
		if err != nil {
			panic("unable to listen to " + options.Bind)
		}
	} else {
		fmt.Println("Starting relay mode at", options.Bind, "forwarding to"+options.Server+"...")
		srv, _ := NewClient(options.Server)
		err := srv.ListenAndServe("tcp", options.Bind)
		if err != nil {
			panic("unable to listen to " + options.Bind)
		}
	}
}
