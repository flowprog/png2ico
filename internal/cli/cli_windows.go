package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"png2ico/internal/ico"
)

const cliUsage = `PNG to ICO

Usage: png2ico-cli.exe [options] input.png
       png2ico-cli.exe input.png output.ico

If no output path is specified, an ICO with the same name is saved beside the PNG.
Options may appear before or after the PNG path. Quote paths containing spaces.

  -o, --output PATH    Set the ICO output path
  -f, --force          Overwrite an existing ICO (refused by default)
  -h, --help           Show help
  --                  Treat all remaining arguments as file paths

On success, the full ICO path is written to stdout. Errors are written to stderr.
Exit codes: 0 for success or help, 1 for conversion failure, 2 for invalid arguments.
`

type cliOptions struct {
	source, output string
	force, help    bool
}

func parseCLI(args []string) (cliOptions, error) {
	var opts cliOptions
	var paths []string
	outputSet := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			opts.help = true
		case "-f", "--force":
			opts.force = true
		case "-o", "--output":
			if outputSet || i+1 >= len(args) {
				return opts, errors.New("-o/--output requires a path and may only be specified once")
			}
			i++
			opts.output, outputSet = args[i], true
		case "--":
			paths = append(paths, args[i+1:]...)
			i = len(args)
		default:
			if strings.HasPrefix(arg, "--output=") || strings.HasPrefix(arg, "-o=") {
				if outputSet {
					return opts, errors.New("the output path may only be specified once")
				}
				_, opts.output, _ = strings.Cut(arg, "=")
				outputSet = true
			} else if strings.HasPrefix(arg, "-") {
				return opts, fmt.Errorf("unknown option: %s", arg)
			} else {
				paths = append(paths, arg)
			}
		}
	}
	if opts.help {
		return opts, nil
	}
	if len(paths) < 1 || len(paths) > 2 || cleanPath(paths[0]) == "" {
		return opts, errors.New("specify one PNG path and, optionally, one ICO output path")
	}
	if len(paths) == 2 {
		if outputSet {
			return opts, errors.New("specify the output using either -o or a second path, not both")
		}
		opts.output, outputSet = paths[1], true
	}
	if outputSet && cleanPath(opts.output) == "" {
		return opts, errors.New("the ICO output path must not be empty")
	}
	opts.source = paths[0]
	return opts, nil
}

// CLI conversion never opens a window or an interactive overwrite dialog.
func Run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseCLI(args)
	if err != nil {
		fmt.Fprintln(stderr, "Argument error: "+err.Error())
		fmt.Fprintln(stderr, "Use --help for usage information.")
		return 2
	}
	if opts.help {
		fmt.Fprint(stdout, cliUsage)
		return 0
	}
	output, err := ico.Convert(opts.source, opts.output, opts.force)
	if err != nil {
		if errors.Is(err, ico.ErrOutputExists) {
			fmt.Fprintf(stderr, "Conversion failed: %s already exists; use -f/--force to overwrite it.\n", output)
		} else {
			fmt.Fprintln(stderr, "Conversion failed: "+err.Error())
		}
		return 1
	}
	if _, err := fmt.Fprintln(stdout, output); err != nil {
		fmt.Fprintln(stderr, "ICO saved, but unable to write to stdout: "+err.Error())
		return 1
	}
	return 0
}

func cleanPath(path string) string {
	return strings.Trim(strings.TrimSpace(path), "\"")
}
