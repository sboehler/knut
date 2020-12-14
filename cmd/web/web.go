// Copyright 2020 Silvio Böhler
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "web",
		Short: "start the web interface",
		Long:  ``,

		RunE: run,
	}
	cmd.Flags().Uint16P("port", "p", 4000, "port to listen on")
	cmd.Flags().StringP("address", "a", "localhost", "addres to listen on")
	return &cmd
}

func run(cmd *cobra.Command, args []string) (errors error) {
	port, err := cmd.Flags().GetUint16("port")
	if err != nil {
		return err
	}
	address, err := cmd.Flags().GetString("address")
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%d", address, port)

	ctx, cancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()
	mux.Handle("/api", Handler())
	srv := &http.Server{
		Addr:        addr,
		Handler:     mux,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}
	srv.RegisterOnShutdown(cancel)

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("error starting server: %v", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill)

	<-stop
	fmt.Println("shutting down...")

	go func() {
		<-stop
		fmt.Println("terminating...")
		os.Exit(1)
	}()

	gracefullCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(gracefullCtx); err != nil {
		return fmt.Errorf("error shuttng down: %v", err)
	}
	fmt.Println("server stopped")
	return nil
}

// Handler is an example handler
func Handler() http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")

		body, err := json.Marshal(map[string]interface{}{
			"foobar": "barfoo",
		})

		if err != nil {
			res.WriteHeader(500)
			return
		}

		res.WriteHeader(200)
		res.Write(body)
	}
}
