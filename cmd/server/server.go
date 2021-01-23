package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/sboehler/knut/lib/balance"
	"github.com/sboehler/knut/lib/date"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/ledger"
	"github.com/sboehler/knut/lib/report"
	"github.com/sboehler/knut/lib/table"
	"github.com/spf13/cobra"
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "web",
		Short: "start the knut web frontend",
		Long:  `start the knut web frontend`,

		Args: cobra.ExactValidArgs(1),

		Run: run,
	}
	cmd.Flags().Int16P("port", "p", 9001, "port")
	cmd.Flags().StringP("address", "a", "localhost", "listen address")
	return cmd
}

func run(cmd *cobra.Command, args []string) {
	if err := execute(cmd, args); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func execute(cmd *cobra.Command, args []string) error {
	port, err := cmd.Flags().GetInt16("port")
	if err != nil {
		return err
	}
	address, err := cmd.Flags().GetString("address")
	if err != nil {
		return err
	}
	var (
		s   = &Server{File: args[0]}
		srv = http.Server{
			Handler: s,
			Addr:    fmt.Sprintf("%s:%d", address, port),
		}
	)
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

// Server handles HTTP.
type Server struct {
	File string
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var ppl = buildPipeline(s.File)
	if err := processPipeline(w, ppl); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type pipeline struct {
	Journal        journal.Journal
	LedgerFilter   ledger.Filter
	BalanceBuilder balance.Builder
	ReportBuilder  report.Builder
	ReportRenderer report.Renderer
	TextRenderer   table.TextRenderer
}

func buildPipeline(file string) *pipeline {
	var (
		journal = journal.Journal{
			File: file,
		}

		period = date.Monthly

		ledgerFilter = ledger.Filter{
			// CommoditiesFilter: filterCommoditiesRegex,
			// AccountsFilter:    filterAccountsRegex,
		}

		balanceBuilder = balance.Builder{
			// From:      from,
			// To:        to,
			Period: &period,
			// Last:      last,
			// Valuation: valuation,
			// Close:     close,
			// Diff:      diff,
		}

		reportBuilder = report.Builder{
			// Value:    valuation != nil,
			// Collapse: collapse,
		}

		reportRenderer = report.Renderer{
			Commodities: true, // showCommodities || valuation == nil,
		}

		tableRenderer = table.TextRenderer{
			// Color:     color,
			// Thousands: thousands,
			// Round:     digits,
		}
	)

	return &pipeline{
		Journal:        journal,
		LedgerFilter:   ledgerFilter,
		BalanceBuilder: balanceBuilder,
		ReportBuilder:  reportBuilder,
		ReportRenderer: reportRenderer,
		TextRenderer:   tableRenderer,
	}
}

func processPipeline(w io.Writer, ppl *pipeline) error {
	l, err := ledger.FromDirectives(ppl.LedgerFilter, ppl.Journal.Parse())
	if err != nil {
		return err
	}
	b, err := ppl.BalanceBuilder.Build(l)
	if err != nil {
		return err
	}
	r, err := ppl.ReportBuilder.Build(b)
	if err != nil {
		return err
	}
	return ppl.TextRenderer.Render(ppl.ReportRenderer.Render(r), w)
}
