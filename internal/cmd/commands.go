package cmd

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/mitchellh/cli"
	"github.com/ryanuber/columnize"
	"github.com/umbracle/eth2-validator/internal/cmd/server"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"google.golang.org/grpc"
)

// Commands returns the cli commands
func Commands() map[string]cli.CommandFactory {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	meta := &Meta{
		UI: ui,
	}

	return map[string]cli.CommandFactory{
		"server": func() (cli.Command, error) {
			return &server.Command{
				UI: ui,
			}, nil
		},
		"validator": func() (cli.Command, error) {
			return &Validators{
				UI: ui,
			}, nil
		},
		"validator list": func() (cli.Command, error) {
			return &ValidatorsList{
				Meta: meta,
			}, nil
		},
		"validator duties": func() (cli.Command, error) {
			return &ValidatorDutiesCommand{
				Meta: meta,
			}, nil
		},
		"duty": func() (cli.Command, error) {
			return &DutyCommand{
				UI: ui,
			}, nil
		},
		"duty list": func() (cli.Command, error) {
			return &DutyListCommand{
				Meta: meta,
			}, nil
		},
		"version": func() (cli.Command, error) {
			return &VersionCommand{
				UI: ui,
			}, nil
		},
	}
}

type Meta struct {
	UI   cli.Ui
	addr string
}

func (m *Meta) FlagSet(n string) *flag.FlagSet {
	f := flag.NewFlagSet(n, flag.ContinueOnError)
	f.StringVar(&m.addr, "address", "localhost:4002", "Address of the http api")
	return f
}

// Conn returns a grpc connection
func (m *Meta) Conn() (proto.ValidatorServiceClient, error) {
	conn, err := grpc.Dial(m.addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}
	clt := proto.NewValidatorServiceClient(conn)
	return clt, nil
}

func formatList(in []string) string {
	columnConf := columnize.DefaultConfig()
	columnConf.Empty = "<none>"
	return columnize.Format(in, columnConf)
}

func formatKV(in []string) string {
	columnConf := columnize.DefaultConfig()
	columnConf.Empty = "<none>"
	columnConf.Glue = " = "
	return columnize.Format(in, columnConf)
}
