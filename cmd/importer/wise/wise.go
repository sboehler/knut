package wise

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sboehler/knut/cmd/flags"
	"github.com/sboehler/knut/cmd/importer"
	"github.com/sboehler/knut/lib/amounts"
	"github.com/sboehler/knut/lib/journal"
	"github.com/sboehler/knut/lib/model"
	"github.com/sboehler/knut/lib/model/posting"
	"github.com/sboehler/knut/lib/model/registry"
	"github.com/sboehler/knut/lib/model/transaction"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

type column int

const (
	cID column = iota
	cStatus
	cDirection
	cCreatedOn
	cFinishedOn
	cSourceFeeAmount
	cSourceFeeCurrency
	cTargetFeeAmount
	cTargetFeeCurrency
	cSourceName
	cSourceAmountAfterFees
	cSourceCurrency
	cTargetName
	cTargetAmountAfterFees
	cTargetCurrency
	cExchangeRate
	cReference
	cBatch
)

// CreateCmd creates the command.
func CreateCmd() *cobra.Command {
	var r runner
	cmd := &cobra.Command{
		Use:   "com.wise",
		Short: "Import Wise CSV account statements",
		Long:  `Download transactions as a CSV file. Make sure the app language is set to English.`,

		RunE: r.run,
	}
	r.setupFlags(cmd)
	return cmd
}

func init() {
	importer.RegisterImporter(CreateCmd)
}

type runner struct {
	account, feeAccount, tradingAccount flags.AccountFlag
}

func (r *runner) setupFlags(cmd *cobra.Command) {
	cmd.Flags().VarP(&r.account, "account", "a", "account name")
	cmd.Flags().VarP(&r.feeAccount, "fee", "f", "fee account name")
	cmd.Flags().VarP(&r.tradingAccount, "trading", "t", "account name of the trading gain / loss account")
	cmd.MarkFlagRequired("account")
	cmd.MarkFlagRequired("fee")
}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	var (
		ctx = registry.New()
		f   *bufio.Reader
		err error
	)
	j := journal.New(ctx)
	for _, path := range args {
		if f, err = flags.OpenFile(path); err != nil {
			return err
		}
		p := parser{
			reader:  csv.NewReader(f),
			journal: j,
		}
		if p.account, err = r.account.Value(ctx.Accounts()); err != nil {
			return err
		}
		if p.feeAccount, err = r.feeAccount.Value(ctx.Accounts()); err != nil {
			return err
		}
		if p.tradingAccount, err = r.tradingAccount.Value(ctx.Accounts()); err != nil {
			return err
		}
		if err = p.parse(); err != nil {
			return err
		}
	}
	out := bufio.NewWriter(cmd.OutOrStdout())
	defer out.Flush()
	return journal.Print(out, j)
}

type parser struct {
	reader                              *csv.Reader
	account, feeAccount, tradingAccount *model.Account
	journal                             *journal.Journal
	balance                             amounts.Amounts
}

func (p *parser) parse() error {
	p.reader.TrimLeadingSpace = true
	p.reader.Comma = ','
	p.reader.FieldsPerRecord = 18
	p.balance = make(amounts.Amounts)

	if err := p.parseHeader(); err != nil {
		return err
	}
	for {
		if err := p.parseBooking(); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}

func (p *parser) parseHeader() error {
	r, err := p.reader.Read()
	if err != nil {
		return err
	}
	header := []string{
		"ID",
		"Status",
		"Direction",
		"Created on",
		"Finished on",
		"Source fee amount",
		"Source fee currency",
		"Target fee amount",
		"Target fee currency",
		"Source name",
		"Source amount (after fees)",
		"Source currency",
		"Target name",
		"Target amount (after fees)",
		"Target currency",
		"Exchange rate",
		"Reference",
		"Batch",
	}
	for i, want := range header {
		if r[i] != want {
			return fmt.Errorf("invalid column name: got %s, want %s", r[i], want)
		}
	}
	return nil
}

func (p *parser) parseFee(bs posting.Builders, feeAmount string, feeCurrency string) (posting.Builders, error) {
	if len(feeCurrency) > 0 {
		amount, err := decimal.NewFromString(feeAmount)
		if err != nil || amount.IsZero() {
			return bs, err
		}
		commodity := p.journal.Registry.Commodities().MustGet(feeCurrency)
		bs = append(bs, posting.Builder{
			Credit:    p.account,
			Debit:     p.feeAccount,
			Quantity:  amount,
			Commodity: commodity,
		})
	}
	return bs, nil
}

func (p *parser) parseBooking() error {
	r, err := p.reader.Read()
	if err != nil {
		return err
	}
	date, err := time.Parse("2006-01-02", r[cCreatedOn][:10])
	if err != nil {
		return fmt.Errorf("invalid started date in row %v: %w", r, err)
	}

	if r[cStatus] == "CANCELLED" {
		return nil
	}

	var bookings posting.Builders
	bookings, err = p.parseFee(bookings, r[cSourceFeeAmount], r[cSourceFeeCurrency])
	if err != nil {
		return err
	}
	bookings, err = p.parseFee(bookings, r[cTargetFeeAmount], r[cTargetFeeCurrency])
	if err != nil {
		return err
	}
	sourceAmount, err := decimal.NewFromString(r[cSourceAmountAfterFees])
	if err != nil {
		return err
	}
	targetAmount, err := decimal.NewFromString(r[cTargetAmountAfterFees])
	if err != nil {
		return err
	}
	sourceCommodity := p.journal.Registry.Commodities().MustGet(r[cSourceCurrency])
	targetCommodity := p.journal.Registry.Commodities().MustGet(r[cTargetCurrency])

	repl := strings.NewReplacer("-", " ", "_", " ")

	if r[cSourceCurrency] != r[cTargetCurrency] {
		bookings = append(bookings,
			posting.Builder{
				Credit:    p.account,
				Debit:     p.tradingAccount,
				Quantity:  sourceAmount,
				Commodity: sourceCommodity,
			},
			posting.Builder{
				Credit:    p.tradingAccount,
				Debit:     p.account,
				Quantity:  targetAmount,
				Commodity: targetCommodity,
			},
		)
		t := transaction.Builder{
			Date:        date,
			Description: fmt.Sprintf("%s / convert %s %s to %s %s", repl.Replace(r[cID]), sourceAmount.String(), sourceCommodity.String(), targetAmount.String(), targetCommodity.String()),
			Postings:    bookings.Build(),
		}.Build()
		p.journal.AddTransaction(t)
		bookings = nil
		switch r[cDirection] {
		case "OUT":
			bookings = append(bookings, posting.Builder{
				Credit:    p.account,
				Debit:     p.journal.Registry.Accounts().TBDAccount(),
				Quantity:  targetAmount,
				Commodity: targetCommodity,
			})
		case "IN":
			bookings = append(bookings, posting.Builder{
				Credit:    p.journal.Registry.Accounts().TBDAccount(),
				Debit:     p.account,
				Quantity:  targetAmount,
				Commodity: targetCommodity,
			})
		case "NEUTRAL":
			return nil
		default:
			return fmt.Errorf("invalid direction: %s", r[cDirection])
		}
	} else {
		switch r[cDirection] {
		case "OUT":
			bookings = append(bookings, posting.Builder{
				Credit:    p.account,
				Debit:     p.journal.Registry.Accounts().TBDAccount(),
				Quantity:  sourceAmount,
				Commodity: sourceCommodity,
			})
		case "IN":
			bookings = append(bookings, posting.Builder{
				Credit:    p.journal.Registry.Accounts().TBDAccount(),
				Debit:     p.account,
				Quantity:  sourceAmount,
				Commodity: sourceCommodity,
			})
		case "NEUTRAL":
			return nil
		default:
			return fmt.Errorf("invalid direction: %s", r[cDirection])
		}
	}

	p.journal.AddTransaction(transaction.Builder{
		Date:        date,
		Description: fmt.Sprintf("%s / %s", repl.Replace(r[cID]), r[cTargetName]),
		Postings:    bookings.Build(),
	}.Build())
	return nil

}
