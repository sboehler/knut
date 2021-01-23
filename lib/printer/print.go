package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/sboehler/knut/lib/ledger"
)

// PrintDirective prints a directive to the given Writer.
func PrintDirective(w io.Writer, directive interface{}) (n int, err error) {
	switch d := directive.(type) {
	case *ledger.Transaction:
		return printTransaction(w, d)
	case *ledger.Open:
		return printOpen(w, d)
	case *ledger.Close:
		return printClose(w, d)
	case *ledger.Assertion:
		return printAssertion(w, d)
	case *ledger.Include:
		return printInclude(w, d)
	case *ledger.Price:
		return printPrice(w, d)
	case *ledger.Accrual:
		return printAccrual(w, d)
	case *ledger.Value:
		return printValue(w, d)
	}
	return 0, fmt.Errorf("unknown directive: %v", directive)
}

func printTransaction(w io.Writer, t *ledger.Transaction) (n int, err error) {
	c, err := fmt.Fprintf(w, "%s \"%s\"", t.Date.Format("2006-01-02"), t.Description)
	n += c
	if err != nil {
		return n, err
	}
	for _, tag := range t.Tags {
		c, err := fmt.Fprintf(w, " %s", tag)
		n += c
		if err != nil {
			return n, err
		}
	}
	c, err = io.WriteString(w, "\n")
	n += c
	if err != nil {
		return n, err
	}
	for _, po := range t.Postings {
		d, err := printPosting(w, po)
		n += d
		if err != nil {
			return n, err
		}
		c, err = io.WriteString(w, "\n")
		n += c
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func printAccrual(w io.Writer, a *ledger.Accrual) (n int, err error) {
	c, err := fmt.Fprintf(w, "@accrue %s %s %s %s\n", a.Period, a.T0.Format("2006-01-02"), a.T1.Format("2006-01-02"), a.Account)
	n += c
	if err != nil {
		return n, err
	}
	c, err = printTransaction(w, a.Transaction)
	return n + c, err
}

func printPosting(w io.Writer, t *ledger.Posting) (int, error) {
	var n int
	c, err := fmt.Fprintf(w, "%s %s %s %s", t.Credit.RightPad(), t.Debit.RightPad(), leftPad(10, t.Amount.Amount().String()), t.Commodity)
	n += c
	if err != nil {
		return n, err
	}
	if t.Lot != nil {
		c, err = io.WriteString(w, " ")
		n += c
		if err != nil {
			return n, err
		}
		d, err := printLot(w, t.Lot)
		n += d
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func printLot(w io.Writer, l *ledger.Lot) (int, error) {
	var n int
	c, err := fmt.Fprintf(w, "{ %g %s, %s ", l.Price, l.Commodity, l.Date.Format("2006-01-02"))
	n += c
	if err != nil {
		return n, err
	}
	if len(l.Label) > 0 {
		c, err = fmt.Fprintf(w, "%s ", l.Label)
		n += c
		if err != nil {
			return n, err
		}
	}
	c, err = io.WriteString(w, "}")
	n += c
	if err != nil {
		return n, err
	}
	return n, nil
}

func printOpen(w io.Writer, o *ledger.Open) (int, error) {
	return fmt.Fprintf(w, "%s open %s", o.Date.Format("2006-01-02"), o.Account)
}

func printClose(w io.Writer, c *ledger.Close) (int, error) {
	return fmt.Fprintf(w, "%s close %s", c.Date.Format("2006-01-02"), c.Account)
}

func printPrice(w io.Writer, pr *ledger.Price) (int, error) {
	return fmt.Fprintf(w, "%s price %s %s %s", pr.Date.Format("2006-01-02"), pr.Commodity, pr.Price, pr.Target)
}

func printInclude(w io.Writer, i *ledger.Include) (int, error) {
	return fmt.Fprintf(w, "include \"%s\"", i.Path)
}

func printAssertion(w io.Writer, a *ledger.Assertion) (int, error) {
	return fmt.Fprintf(w, "%s balance %s %s %s", a.Date.Format("2006-01-02"), a.Account, a.Amount, a.Commodity)
}

func printValue(w io.Writer, v *ledger.Value) (int, error) {
	return fmt.Fprintf(w, "%s value %s %s %s", v.Date.Format("2006-01-02"), v.Account, v.Amount, v.Commodity)
}

// PrintLedger prints a Ledger.
func PrintLedger(w io.Writer, l ledger.Ledger) (int, error) {
	var n int
	for _, day := range l {
		for _, pr := range day.Prices {
			if err := writeLn(w, pr, &n); err != nil {
				return n, err
			}
		}
		for _, o := range day.Openings {
			if err := writeLn(w, o, &n); err != nil {
				return n, err
			}
		}
		for _, t := range day.Transactions {
			if err := writeLn(w, t, &n); err != nil {
				return n, err
			}
		}
		for _, v := range day.Values {
			if err := writeLn(w, v, &n); err != nil {
				return n, err
			}
		}
		for _, a := range day.Assertions {
			if err := writeLn(w, a, &n); err != nil {
				return n, err
			}
		}
		for _, c := range day.Closings {
			if err := writeLn(w, c, &n); err != nil {
				return n, err
			}
		}
	}
	return n, nil
}

func writeLn(w io.Writer, d interface{}, count *int) error {
	c, err := PrintDirective(w, d)
	*count += c
	if err != nil {
		return err
	}
	c, err = io.WriteString(w, "\n")
	*count += c
	if err != nil {
		return err
	}
	return nil
}

func leftPad(n int, s string) string {
	if len(s) > n {
		return s
	}
	var b strings.Builder
	for i := 0; i < n-len(s); i++ {
		b.WriteRune(' ')
	}
	b.WriteString(s)
	return b.String()
}
