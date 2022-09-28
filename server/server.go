package server

import (
	"fmt"
	"io"

	pb "github.com/sboehler/knut/server/proto"
)

func Test(w io.Writer) error {
	p := pb.HelloRequest{
		Name: "Foobar",
	}
	fmt.Fprintln(w, p.String())
	return nil
}
