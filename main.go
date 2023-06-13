package main

import (
	"egg/socks5"
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
)

type ServerCMD struct {
	Bind string `short:"b" long:"bind" default:":8585" description:"Binding address, where should server listen to. default :5858"`
}

func (s *ServerCMD) Execute(_ []string) error {
	// run server mode, ie open http server and listen to incoming requests from internet
	fmt.Printf("Starting server at %s ...\n", s.Bind)
	srv := NewServer()
	err := srv.ListenAndServe(s.Bind)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
		return err
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		return err
	}
	return nil
}

var serverCMD ServerCMD

type ClientCMD struct {
	Bind     string `short:"b" long:"bind" default:":8585" description:"Binding address, where should socks proxy server listen to. default :5858"`
	Server   string `short:"s" long:"server" default:"ws://127.0.0.1:8585/ws" description:"Remote websocket server address, it should starts with ws or wss and ends with ws path ex. wss://example.com/ws"`
	Upath    string `short:"u" long:"upload" description:"Uploading part of connections will be forwarded to <ip>:<port>. for using it you must setup relay server first, then provide this argument with address of forwarding server. ex. example.com:5858"`
	Insecure bool   `short:"k" long:"insecure" description:"Allow to connect to insecure endpoints default: false. (not recommended)"`
}

func (c *ClientCMD) Execute(_ []string) error {
	// run client mode, ie open http server and listen to incoming requests from internet
	fmt.Printf("Starting client at %s ...\n", c.Bind)
	var srv *socks5.Server
	if c.Upath != "" {
		u, _ := url.Parse(c.Server)
		RelayAddress = c.Upath
		RelayAddressToReplace = u.Host
		srv, _ = NewClient(c.Server, true)
	} else {
		srv, _ = NewClient(c.Server, false)
	}
	err := srv.ListenAndServe("tcp", c.Bind)
	if err != nil {
		fmt.Printf("unable to listen to %s\n", c.Bind)
		return err
	}
	return nil
}

var clientCMD ClientCMD

type RelayCMD struct {
	Bind    string `short:"b" long:"bind" default:":8585" description:"Binding address, where should I listen to. client default: :8585, server default: :5858"`
	Forward string `short:"f" long:"forward" description:"Specifies where should incoming tcp connections getting forward to <ip>:<port>"`
}

func (r *RelayCMD) Execute(_ []string) error {
	// run client mode, ie open http server and listen to incoming requests from internet
	fmt.Printf("Starting realay at %s forwarding to %s...\n", r.Bind, r.Forward)
	return NewRelay(r.Bind, r.Forward)
}

var relayCMD RelayCMD

var parser = flags.NewParser(nil, flags.Default)

func init() {
	_, _ = parser.AddCommand("server",
		"Server mode",
		"It set's up a server a serve to incoming proxified connections from client and deliver them to actual dest",
		&serverCMD)

	_, _ = parser.AddCommand("client",
		"Client mode",
		"It set's up a socks5 proxy server and forward its incoming connections to remote server",
		&clientCMD)

	_, _ = parser.AddCommand("relay",
		"Relay mode",
		"It set's up a relay server and forward's all incoming connections to a destination address",
		&relayCMD)
}

func cleanup() {
	fmt.Println("Cleanup...")
}

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(0)
	}()

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
}
