package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/k4k3ru-hub/cli/go"
	coinbaseexchange "github.com/k4k3ru-hub/coinbase-exchange/go"
	"github.com/k4k3ru-hub/coinbase-exchange/go/marketdata"
)

const version = "1.0.0"

type serverTimeGetter interface {
	GetServerTime(context.Context) (*marketdata.Time, error)
}

func main() {
	restClient, err := coinbaseexchange.NewRESTClient(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	commandLine, err := newCLI(restClient.MarketData())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := commandLine.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCLI(timeGetter serverTimeGetter) (*cli.CLI, error) {
	if timeGetter == nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: time_getter=null")
	}

	commandLine := cli.NewCLI(nil)
	commandLine.SetVersion(version)
	if err := commandLine.Root().AddDefaultConfigOption(); err != nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: %w", err)
	}

	restCommand := cli.NewCommand("rest")
	restCommand.SetUsage("Coinbase Exchange public REST API commands.")
	if err := commandLine.Root().AddCommand(restCommand); err != nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: %w", err)
	}

	timeCommand := cli.NewCommand("time")
	timeCommand.SetUsage("Get Coinbase Exchange server time.")
	timeCommand.SetAction(func(ctx *cli.Context) error {
		result, err := timeGetter.GetServerTime(context.Background())
		if err != nil {
			return fmt.Errorf("failed to run coinbase exchange rest time command: %w", err)
		}
		encoder := json.NewEncoder(ctx.Output())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("failed to run coinbase exchange rest time command: failed to output result: %w", err)
		}
		return nil
	})
	if err := restCommand.AddCommand(timeCommand); err != nil {
		return nil, fmt.Errorf("failed to create coinbase exchange cli: %w", err)
	}

	return commandLine, nil
}
