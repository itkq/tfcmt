package controller

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/suzuki-shunsuke/go-error-with-exit-code/ecerror"
	"github.com/suzuki-shunsuke/tfcmt/v4/pkg/config"
	"github.com/suzuki-shunsuke/tfcmt/v4/pkg/mask"
	"github.com/suzuki-shunsuke/tfcmt/v4/pkg/notifier"
	"github.com/suzuki-shunsuke/tfcmt/v4/pkg/platform"
)

// Plan sends the notification with notifier
func (c *Controller) Plan(ctx context.Context, logger *slog.Logger, command Command) error {
	if command.Cmd == "" {
		return errors.New("no command specified")
	}
	if err := platform.Complement(&c.Config); err != nil {
		return err
	}

	if err := c.Config.Validate(); err != nil {
		return err
	}

	ntf, err := c.getPlanNotifier(ctx)
	if err != nil {
		return err
	}

	if ntf == nil {
		return errors.New("no notifier specified at all")
	}

	cmd := exec.CommandContext(ctx, command.Cmd, command.Args...) //nolint:gosec
	cmd.Stdin = os.Stdin
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	combinedOutput := &bytes.Buffer{}
	uncolorizedStdout := colorable.NewNonColorable(stdout)
	uncolorizedStderr := colorable.NewNonColorable(stderr)
	uncolorizedCombinedOutput := colorable.NewNonColorable(combinedOutput)
	cmd.Stdout = io.MultiWriter(mask.NewWriter(os.Stdout, c.Config.Masks), uncolorizedStdout, uncolorizedCombinedOutput)
	cmd.Stderr = io.MultiWriter(mask.NewWriter(os.Stderr, c.Config.Masks), uncolorizedStderr, uncolorizedCombinedOutput)
	setCancel(cmd)
	_ = cmd.Run()

	combined := combinedOutput.String()
	exitCode := cmd.ProcessState.ExitCode()

	notifyErr := ntf.Plan(ctx, logger, &notifier.ParamExec{
		Stdout:         stdout.String(),
		Stderr:         stderr.String(),
		CombinedOutput: combined,
		CIName:         c.Config.CI.Name,
		ExitCode:       exitCode,
	})

	if c.Config.Terraform.Plan.FailWarning.Enable && exitCode == 0 && notifyErr == nil {
		result := c.Parser.Parse(combined)
		if hasUnignoredWarnings(result.Warning, c.Config.Terraform.Plan.FailWarning.IgnoreWarnings, c.Config.Vars["target"]) {
			return ecerror.Wrap(errors.New("terraform plan has warnings"), 1)
		}
	}

	return ecerror.Wrap(notifyErr, exitCode)
}

const waitDelay = 1000 * time.Hour

func hasUnignoredWarnings(warning string, ignoreWarnings []config.IgnoreWarning, target string) bool {
	warning = strings.TrimSpace(warning)
	if warning == "" {
		return false
	}
	for _, iw := range ignoreWarnings {
		if iw.TargetRegexp != nil && !iw.TargetRegexp.MatchString(target) {
			continue
		}
		if iw.WarningRegexp != nil && iw.WarningRegexp.MatchString(warning) {
			return false
		}
	}
	return true
}

func setCancel(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt) //nolint:wrapcheck
	}
	cmd.WaitDelay = waitDelay
}
