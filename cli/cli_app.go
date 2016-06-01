package cli

import (
	"github.com/urfave/cli"
	"github.com/gruntwork-io/terragrunt/config"
	"github.com/gruntwork-io/terragrunt/locks"
	"fmt"
	"strings"
	"github.com/gruntwork-io/terragrunt/shell"
	"github.com/gruntwork-io/terragrunt/util"
)

const CUSTOM_USAGE_TEXT = `DESCRIPTION:
   {{.Name}} - {{.UsageText}}

USAGE:
   {{.Usage}}

COMMANDS:
   apply                Acquire a lock and run 'terraform apply'
   destroy              Acquire a lock and run 'terraform destroy'
   release-lock         Release a lock that is left over from some previous command
   *                    Terragrunt forwards all other commands directly to Terraform
{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}
VERSION:
   {{.Version}}{{if len .Authors}}

AUTHOR(S):
   {{range .Authors}}{{.}}{{end}}
   {{end}}
`

func CreateTerragruntCli(version string) *cli.App {
	cli.AppHelpTemplate = CUSTOM_USAGE_TEXT

	app := cli.NewApp()

	app.Name = "terragrunt"
	app.Author = "Gruntwork <www.gruntwork.io>"
	app.Version = version
	app.Action = runApp
	app.Usage = "terragrunt <COMMAND>"
	app.UsageText = `Terragrunt is a thin wrapper for the Terraform client that provides a distributed locking
   mechanism which allows multiple people to collaborate on the same Terraform state without overwriting each other's
   changes. Terragrunt currently uses Amazon's DynamoDB to acquire and release locks. For documentation, see
   https://github.com/gruntwork-io/terragrunt/.

   Terragrunt supports all the same commands as Terraform (e.g. plan, apply, destroy, etc). However, for the apply and
   destroy commands, it will first acquire a lock, then run the command with Terraform, and then release the lock.`

	return app
}

func runApp(cliContext *cli.Context) error {
	args := cliContext.Args()
	switch args.First() {
	case "apply", "destroy": return runTerraformCommandWithLock(cliContext)
	case "release-lock": return releaseLockCommand(cliContext)
	default: return runTerraformCommand(cliContext)
	}
}

func runTerraformCommandWithLock(cliContext *cli.Context) error {
	lock, err := config.GetLockForConfig()
	if err != nil {
		return err
	}

	return locks.WithLock(lock, func() error {
		return runTerraformCommand(cliContext)
	})
}

func runTerraformCommand(cliContext *cli.Context) error {
	util.Logger.Printf("Running command: terraform %s", strings.Join(cliContext.Args(), " "))
	return shell.RunShellCommand("terraform", cliContext.Args()...)
}

func releaseLockCommand(cliContext *cli.Context) error {
	lock, err := config.GetLockForConfig()
	if err != nil {
		return err
	}

	proceed, err := shell.PromptUserForYesNo(fmt.Sprintf("Are you sure you want to release %s?", lock))
	if err != nil {
		return err
	}

	if proceed {
		return lock.ReleaseLock()
	} else {
		return nil
	}
}
