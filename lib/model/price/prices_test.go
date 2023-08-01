package price

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sboehler/knut/lib/model/commodity"
	"github.com/sboehler/knut/lib/model/registry"

	"github.com/shopspring/decimal"
)

func TestPrices(t *testing.T) {
	reg := registry.New()
	com1 := reg.Commodity("COM1")
	com2 := reg.Commodity("COM2")

	tests := []struct {
		desc  string
		input []*Price
		want  Prices
	}{
		{
			desc: "case 1",
			input: []*Price{
				{
					Commodity: com1,
					Target:    com2,
					Price:     decimal.RequireFromString("4.0"),
				},
			},
			want: Prices{
				com2: {
					com1: decimal.RequireFromString("4.0"),
				},
				com1: {
					com2: decimal.RequireFromString("0.25"),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := make(Prices)
			for _, in := range test.input {
				got.Insert(in.Commodity, in.Price, in.Target)
			}
			if diff := cmp.Diff(got, test.want); diff != "" {
				t.Fatalf("unexpected diff (+got/-want):\n%s", diff)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	reg := registry.New()
	com1 := reg.Commodity("COM1")
	com2 := reg.Commodity("COM2")
	com3 := reg.Commodity("COM3")

	tests := []struct {
		desc   string
		input  []*Price
		target *commodity.Commodity
		want   NormalizedPrices
	}{
		{
			desc: "case 1",
			input: []*Price{
				{Commodity: com1, Price: decimal.RequireFromString("4.0"), Target: com2},
				{Commodity: com2, Price: decimal.RequireFromString("2.0"), Target: com3},
			},
			target: com1,
			want: NormalizedPrices{
				com1: decimal.RequireFromString("1"),
				com2: decimal.RequireFromString("0.25"),
				com3: decimal.RequireFromString("0.125"),
			},
		},
		{
			desc: "case 2",
			input: []*Price{
				{Commodity: com1, Price: decimal.RequireFromString("4.0"), Target: com2},
				{Commodity: com2, Price: decimal.RequireFromString("2.0"), Target: com3},
			},
			target: com2,
			want: NormalizedPrices{
				com1: decimal.RequireFromString("4"),
				com2: decimal.RequireFromString("1"),
				com3: decimal.RequireFromString("0.5"),
			},
		},
		{
			desc: "case 3",
			input: []*Price{
				{Commodity: com1, Price: decimal.RequireFromString("4.0"), Target: com2},
				{Commodity: com2, Price: decimal.RequireFromString("2.0"), Target: com3},
			},
			target: com3,
			want: NormalizedPrices{
				com1: decimal.RequireFromString("8"),
				com2: decimal.RequireFromString("2"),
				com3: decimal.RequireFromString("1"),
			},
		},
		{
			desc: "case 4",
			input: []*Price{
				{Commodity: com1, Price: decimal.RequireFromString("4.0"), Target: com2},
				{Commodity: com3, Price: decimal.RequireFromString("2.0"), Target: com2},
			},
			target: com3,
			want: NormalizedPrices{
				com1: decimal.RequireFromString("2"),
				com2: decimal.RequireFromString("0.5"),
				com3: decimal.RequireFromString("1"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pr := make(Prices)
			for _, in := range test.input {
				pr.Insert(in.Commodity, in.Price, in.Target)
			}

			got := pr.Normalize(test.target)

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("unexpected diff (-want/+got):\n%s", diff)
			}
		})
	}
}
