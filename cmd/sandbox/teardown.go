package sandbox

import (
	"context"
	"fmt"

	sandboxConfig "github.com/flyteorg/flytectl/cmd/config/subcommand/sandbox"

	"github.com/flyteorg/flytectl/pkg/docker"

	"github.com/docker/docker/api/types"
	"github.com/enescakir/emoji"

	cmdCore "github.com/flyteorg/flytectl/cmd/core"
)

const (
	teardownShort = "Teardown will cleanup the sandbox environment"
	teardownLong  = `
Teardown will remove docker container and all the flyte config 
::

 bin/flytectl sandbox teardown 
	

Usage
`
)

func teardownSandboxCluster(ctx context.Context, args []string, cmdCtx cmdCore.CommandContext) error {
	cli, err := docker.GetDockerClient()
	if err != nil {
		return err
	}

	return tearDownSandbox(ctx, cli)
}

func tearDownSandbox(ctx context.Context, cli docker.Docker) error {
	name := "flyte-sandbox"
	if len(sandboxConfig.DefaultConfig.Name) > 0 {
		name = sandboxConfig.DefaultConfig.Name
	}
	c := docker.GetSandbox(ctx, cli, name)
	if c != nil {
		if err := cli.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			return err
		}
	}
	if err := docker.ConfigCleanup(); err != nil {
		fmt.Printf("Config cleanup failed. Which Failed due to %v \n ", err)
	}
	fmt.Printf("%v %v Sandbox cluster is removed successfully. \n", emoji.Broom, emoji.Broom)
	return nil
}
