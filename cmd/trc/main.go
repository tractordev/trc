package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"tractor.dev/toolkit-go/engine"
	"tractor.dev/toolkit-go/engine/cli"
	"tractor.dev/trc/builtin/golang"
	"tractor.dev/trc/builtin/jq"
	"tractor.dev/trc/repl"
)

type ExecHandlerProvider interface {
	ProvideExecHandler(next interp.ExecHandlerFunc) interp.ExecHandlerFunc
}

func main() {
	engine.Run(Main{},
		// gum.Builtin{},
		golang.Builtin{},
		jq.Builtin{},
	)
}

type Main struct {
	ExecHandlers []ExecHandlerProvider
	Runner       *interp.Runner
}

func (m *Main) InitializeCLI(root *cli.Command) {
	var command string

	root.Usage = "trc [script...]"
	root.Flags().StringVar(&command, "c", "", "command to be executed")
	root.Run = func(ctx *cli.Context, args []string) {
		var err error
		m.Runner, err = interp.New(interp.StdIO(ctx, ctx, ctx.Errout()), m.execHandlers())
		exit(err)

		// handle -c command
		if command != "" {
			exit(m.Execute(ctx, strings.NewReader(command), ""))
			return
		}

		// handle source file arguments
		if len(args) > 0 {
			for _, path := range args {
				f, err := os.Open(path)
				exit(err)
				defer f.Close()
				if err := m.Execute(ctx, f, path); err != nil {
					exit(err)
					return
				}
			}
			return
		}

		// handle terminal
		if term.IsTerminal(int(os.Stdin.Fd())) {
			exit(repl.Run(ctx, runner))
			//handleExit(m.RunInteractive(runner, os.Stdin, os.Stdout, os.Stderr))
			return
		}

		// handle non-terminal stdin source
		exit(m.Execute(ctx, ctx, ""))
		return

	}

}

func (m *Main) Execute(ctx context.Context, source io.Reader, name string) (err error) {
	prog, err := syntax.NewParser(syntax.Variant(syntax.LangBats)).Parse(source, name)
	if err != nil {
		return err
	}
	m.Runner.Reset()
	return m.Runner.Run(ctx, prog)
}

func (m *Main) RunInteractive(r *interp.Runner, stdin io.Reader, stdout, stderr io.Writer) (err error) {
	parser := syntax.NewParser()
	fmt.Fprintf(stdout, "$ ")
	var runErr error
	fn := func(stmts []*syntax.Stmt) bool {
		if parser.Incomplete() {
			fmt.Fprintf(stdout, "> ")
			return true
		}
		ctx := context.Background()
		for _, stmt := range stmts {
			runErr = r.Run(ctx, stmt)
			if r.Exited() {
				return false
			}
		}
		fmt.Fprintf(stdout, "$ ")
		return true
	}
	if err := parser.Interactive(stdin, fn); err != nil {
		return err
	}
	return runErr
}

func (m *Main) execHandlers() interp.RunnerOption {
	var handlers []func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc
	for _, h := range m.ExecHandlers {
		handlers = append(handlers, h.ProvideExecHandler)
	}
	return interp.ExecHandlers(handlers...)
}

func exit(err error) {
	if e, ok := interp.IsExitStatus(err); ok {
		os.Exit(int(e))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
