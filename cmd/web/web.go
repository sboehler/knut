// Copyright 2021 Silvio Böhler
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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/sboehler/knut/api"
	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var cmd = cobra.Command{
		Use:   "web",
		Short: "start the knut web frontend",
		Long:  `start the knut web frontend`,

		Args: cobra.ExactValidArgs(1),

		Run: run,
	}
	cmd.Flags().Int16P("port", "p", 9001, "port")
	cmd.Flags().StringP("address", "a", "localhost", "listen address")
	return &cmd
}

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) error {
	var (
		address string
		port    int16
		err     error
	)
	if port, err = cmd.Flags().GetInt16("port"); err != nil {
		return err
	}
	if address, err = cmd.Flags().GetString("address"); err != nil {
		return err
	}

	var handler = http.NewServeMux()
	handler.Handle("/api/", http.StripPrefix("/api", api.New(args[0])))

	var srv = http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf("%s:%d", address, port),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
		}
	}()
	var c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Println("")
	var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	fmt.Fprintln(cmd.ErrOrStderr(), "shutting down")
	return nil

}
