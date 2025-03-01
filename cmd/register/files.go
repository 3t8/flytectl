package register

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/flyteorg/flytectl/cmd/config"

	"github.com/flyteorg/flytestdlib/storage"

	rconfig "github.com/flyteorg/flytectl/cmd/config/subcommand/register"
	cmdCore "github.com/flyteorg/flytectl/cmd/core"
	"github.com/flyteorg/flytectl/pkg/printer"
	"github.com/flyteorg/flytestdlib/logger"
)

const (
	registerFilesShort = "Registers file resources."
	registerFilesLong  = `
Registers all the serialized protobuf files including tasks, workflows and launch plans with default v1 version.

If previously registered entities with v1 version are present, the command will fail immediately on the first such encounter.
::

 flytectl register file  _pb_output/* -d development  -p flytesnacks

As per Flytectl, registration and fast registration mean the same!

In fast registration, the input provided by the user is fast serialized proto generated by pyflyte.
When the user runs pyflyte with --fast flag, then pyflyte creates serialized proto and the source code archive file in the same directory.
Flytectl finds the input file by searching for an archive file whose name starts with "fast" and has .tar.gz extension.
If Flytectl finds any source code in users' input, it considers the registration as fast registration.

SourceUploadPath is an optional flag. By default, Flytectl will create SourceUploadPath from your storage config.
If s3, Flytectl will upload the code base to s3://{{DEFINE_BUCKET_IN_STORAGE_CONFIG}}/fast/{{VERSION}}-fast{{MD5_CREATED_BY_PYFLYTE}.tar.gz}.
::

 flytectl register file  _pb_output/* -d development  -p flytesnacks  --version v2

In case of fast registration, if the SourceUploadPath flag is defined, Flytectl will not use the default directory to upload the source code.
Instead, it will override the destination path on the registration.
::

 flytectl register file  _pb_output/* -d development  -p flytesnacks  --version v2 --SourceUploadPath="s3://dummy/fast"

To register a .tgz or .tar file, use the --archive flag. They can be local or remote files served through http/https.

::

  flytectl register files http://localhost:8080/_pb_output.tar -d development  -p flytesnacks --archive

Using local tgz file:

::

 flytectl register files  _pb_output.tgz -d development  -p flytesnacks --archive

If you wish to continue executing registration on other files by ignoring the errors including the version conflicts, then send the continueOnError flag:

::

 flytectl register file  _pb_output/* -d development  -p flytesnacks --continueOnError

Using short format of continueOnError flag:
::

 flytectl register file  _pb_output/* -d development  -p flytesnacks --continueOnError

Override the default version v1 using version string:
::

 flytectl register file  _pb_output/* -d development  -p flytesnacks --version v2

Changing the o/p format has no effect on the registration. The O/p is currently available only in table format:

::

 flytectl register file  _pb_output/* -d development  -p flytesnacks --continueOnError -o yaml

Override IamRole during registration:

::

 flytectl register file  _pb_output/* -d development  -p flytesnacks --continueOnError --version v2 --assumableIamRole "arn:aws:iam::123456789:role/dummy"

Override Kubernetes service account during registration:

::

 flytectl register file  _pb_output/* -d development  -p flytesnacks --continueOnError --version v2 --k8sServiceAccount "kubernetes-service-account"

Override Output location prefix during registration:

::

 flytectl register file  _pb_output/* -d development  -p flytesnacks --continueOnError --version v2 --outputLocationPrefix "s3://dummy/prefix"

Override Destination dir of source code in container during registration:

::

 flytectl register file  _pb_output/* -d development  -p flytesnacks --continueOnError --version v2 --destinationDirectory "/root" 

Enable schedule for the launchplans part of the serialized protobuf files:

::

 flytectl register file  _pb_output/* -d development  -p flytesnacks --version v2 --enableSchedule
	
Usage
`
	sourceCodeExtension = ".tar.gz"
)

func registerFromFilesFunc(ctx context.Context, args []string, cmdCtx cmdCore.CommandContext) error {
	return Register(ctx, args, config.GetConfig(), cmdCtx)
}

func Register(ctx context.Context, args []string, cfg *config.Config, cmdCtx cmdCore.CommandContext) error {
	var regErr error
	var dataRefs []string

	// Deprecated checks for --k8Service
	deprecatedCheck(ctx, &rconfig.DefaultFilesConfig.K8sServiceAccount, rconfig.DefaultFilesConfig.K8ServiceAccount)

	// getSerializeOutputFiles will return you all proto and  source code compress file in sorted order
	dataRefs, tmpDir, err := GetSerializeOutputFiles(ctx, args, rconfig.DefaultFilesConfig.Archive)
	if err != nil {
		logger.Errorf(ctx, "error while un-archiving files in tmp dir due to %v", err)
		return err
	}
	logger.Infof(ctx, "Parsing file... Total(%v)", len(dataRefs))

	// It will segregate serialize output files in valid proto,Invalid files if have any and source code(In case of fast serialize input files)
	sourceCodePath, validProto, InvalidFiles := segregateSourceAndProtos(dataRefs)

	// If any invalid files provide in input then through an error
	if len(InvalidFiles) > 0 {
		return fmt.Errorf("input package have some invalid files. try to run pyflyte package again %v", InvalidFiles)
	}

	// In case of fast serialize input upload source code to destination bucket
	var uploadLocation storage.DataReference
	if len(sourceCodePath) > 0 {
		logger.Infof(ctx, "Fast Registration detected")
		uploadLocation, err = uploadFastRegisterArtifact(ctx, cfg.Project, cfg.Domain, sourceCodePath, rconfig.DefaultFilesConfig.Version,
			cmdCtx.ClientSet().DataProxyClient(), rconfig.DefaultFilesConfig.DeprecatedSourceUploadPath)
		if err != nil {
			return fmt.Errorf("failed to upload source code from [%v]. Error: %w", sourceCodePath, err)
		}

		logger.Infof(ctx, "Source code successfully uploaded to [%v]", uploadLocation)
	}

	var registerResults []Result
	fastFail := !rconfig.DefaultFilesConfig.ContinueOnError
	for i := 0; i < len(validProto) && !(fastFail && regErr != nil); i++ {
		registerResults, regErr = registerFile(ctx, validProto[i], registerResults, cmdCtx, uploadLocation, *rconfig.DefaultFilesConfig)
	}

	payload, _ := json.Marshal(registerResults)
	registerPrinter := printer.Printer{}
	_ = registerPrinter.JSONToTable(payload, projectColumns)
	if tmpDir != "" {
		if _err := os.RemoveAll(tmpDir); _err != nil {
			logger.Errorf(ctx, "unable to delete temp dir %v due to %v", tmpDir, _err)
			return _err
		}
	}
	return regErr
}
