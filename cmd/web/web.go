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
	var srv = http.Server{
		Handler: api.Handler{File: args[0]},
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
