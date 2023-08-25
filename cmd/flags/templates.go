package flags

import (
	"github.com/sboehler/knut/lib/common/date"
	"github.com/spf13/cobra"
)

type Multiperiod struct {
	period   PeriodFlag
	last     int
	interval IntervalFlags
}

func (mp *Multiperiod) Setup(cmd *cobra.Command) {
	mp.period.Setup(cmd, date.Period{End: date.Today()})
	cmd.Flags().IntVar(&mp.last, "last", 0, "last n periods")
	mp.interval.Setup(cmd, date.Once)
}

func (mp *Multiperiod) Partition(clip date.Period) date.Partition {
	return date.NewPartition(mp.period.Value().Clip(clip), mp.interval.Value(), mp.last)
}
