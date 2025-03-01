package sandbox

import (
	"context"
	"fmt"

	"github.com/enescakir/emoji"
	cmdCore "github.com/flyteorg/flytectl/cmd/core"
	"github.com/flyteorg/flytectl/pkg/docker"
)

const (
	statusShort = "Gets the status of the sandbox environment."
	statusLong  = `
Retrieves the status of the sandbox environment. Currently, Flyte sandbox runs as a local Docker container.

Usage
::

 flytectl sandbox status 

`
)

func sandboxClusterStatus(ctx context.Context, args []string, cmdCtx cmdCore.CommandContext) error {
	cli, err := docker.GetDockerClient()
	if err != nil {
		return err
	}

	return printStatus(ctx, cli)
}

func printStatus(ctx context.Context, cli docker.Docker) error {
	c, err := docker.GetSandbox(ctx, cli)
	if err != nil {
		return err
	}
	if c == nil {
		fmt.Printf("%v no Sandbox found \n", emoji.StopSign)
		return nil
	}
	fmt.Printf("Flyte local sandbox cluster container image [%s] with status [%s] is in state [%s]", c.Image, c.Status, c.State)
	return nil
}
