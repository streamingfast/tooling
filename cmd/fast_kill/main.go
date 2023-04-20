package main

import (
	"context"
	"io"
	"math"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/streamingfast/cli"
	. "github.com/streamingfast/cli"
	"github.com/streamingfast/cli/sflags"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

var zlog, _ = logging.ApplicationLogger("fast_kill", "github.com/streamingfast/tooling/cmd/fast_kill", logging.WithConsoleToStderr())

func main() {
	Run(
		"fast_kill <flags> -- <command> [<argument> ...]",
		"Rapidly call the given <command> and sent SIGINT after defined amount, repeating forever or a number of tiems",
		Description(`
			Sometimes, you want to reproduce a bug that happen occasionnaly when exiting the
			your program.

			The 'fast_kill' command launches the received <command> with <arguments> if any and then
			wait '--wait-before-kill <duration>' (defaults to 5s) then send a SIGINT to the running
			command.

			The '--wait-after-kill <duration>' (defaults to 0) can be used to define the wait time after the command
			fully exited before starting the program again.

			Those steps are repeated forever or the number defines by the flag '--repeat <count>'
			(defaults to <forever>).
		`),
		MinimumNArgs(1),
		Flags(func(flags *pflag.FlagSet) {
			flags.IntP("repeat", "r", -1, "If defined, stop after 5 cycle of start/stop instead of running forever")
			flags.DurationP("wait-before-kill", "b", 5*time.Second, "Defines the wait time after program launched before sending the SIGINT signal")
			flags.DurationP("wait-after-kill", "w", 0, "Defines the wait time after the command fully exited before starting the program again")

			// FIXME: Add support, it's kind of complicated because we cannot use `strings.Contains(...)` directly since
			// we are copying on the fly bytes, so we need a special "matcher" that is going to be able to work across
			// multiple "segment" of output and determine a match across those 2 or more segments.
			// flags.StringP("stop-when-seen", "s", "", "If set, inspect the command's output and stop 'fast_kill' on first encounter")
		}),
		Example(`
			# Wait 7s, kill bash script, repeat forever
			fast_kill --wait-before-kill 7s -- bash -c "echo 'Value'; sleep 1"

			# Wait 5s, kill bash script, repeat 5 times
			fast_kill --repeat 5 -- bash -c "echo 'Value'; sleep 1"
		`),
		Execute(func(cmd *cobra.Command, args []string) error {
			// FIXME: Send error when running Windows, not implemented (cannot send signal)
			repeat := sflags.MustGetInt(cmd, "repeat")
			waitBeforeKill := sflags.MustGetDuration(cmd, "wait-before-kill")
			waitAfterKill := sflags.MustGetDuration(cmd, "wait-after-kill")

			zlog.Debug("starting 'fast_kill'",
				zap.Bool("forever", repeat < 0),
				zap.Int("repeat_count", repeat),
				zap.Duration("wait_before_kill", waitBeforeKill),
				zap.Duration("wait_affter_kill", waitAfterKill),
				zap.Strings("arguments", args),
			)

			command := args[0]
			arguments := args[1:]

			if repeat < 0 {
				zlog.Debug("retrying forever, updating repeat")
				repeat = math.MaxInt
			}

			for repeatCount := 0; ; {
				c := exec.Command(command, arguments...)

				zlog.Debug("starting command through PTY", zap.Stringer("cmd", c))
				ptyFile, err := pty.Start(c)
				cli.NoError(err, "Unable to create PTY")

				// FIXME: What to do with error where program would like to receive data written to terminal,
				// for example 'go run ./cmd/fast_kill fast_kill --wait-before-kill 7s -- bash' would like to
				// received user's input, maybe we do not handle those case.

				zlog.Debug("Starting copy")
				commandContext, cancel := context.WithCancel(context.Background())
				go func() {
					select {
					case <-commandContext.Done():
					case <-time.After(waitBeforeKill):
						cli.NoError(c.Process.Signal(syscall.SIGINT), "Sending SIGINT failed")
					}

					repeatSignalEach := 1 * time.Second
					for {
						select {
						case <-commandContext.Done():
						case <-time.After(repeatSignalEach):
							zlog.Debug("Sending repeated signal since it seems we are not closed yet")
							cli.NoError(c.Process.Signal(syscall.SIGINT), "Sending repeated SIGINT failed")
						}
					}
				}()

				_, err = io.Copy(os.Stdout, ptyFile)
				cli.NoError(err, "Unable to copy command PTY to stdout")

				zlog.Debug("Copy terminated")
				cancel()

				repeatCount++
				if repeat == repeatCount {
					break
				}

				// How should we kill that so that we don't wait forever?
				<-time.After(waitAfterKill)
			}

			return nil
		}),
	)
}
