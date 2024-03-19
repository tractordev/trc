package gum

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/gum/choose"
	"github.com/charmbracelet/gum/completion"
	"github.com/charmbracelet/gum/confirm"
	"github.com/charmbracelet/gum/file"
	"github.com/charmbracelet/gum/format"
	"github.com/charmbracelet/gum/input"
	"github.com/charmbracelet/gum/join"
	"github.com/charmbracelet/gum/man"
	"github.com/charmbracelet/gum/pager"
	"github.com/charmbracelet/gum/spin"
	"github.com/charmbracelet/gum/style"
	"github.com/charmbracelet/gum/table"
	"github.com/charmbracelet/gum/write"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"mvdan.cc/sh/v3/interp"
)

var (
	// Version contains the application version number. It's set via ldflags
	// when building.
	Version = ""

	// CommitSHA contains the SHA of the commit that this application was built
	// against. It's set via ldflags when building.
	CommitSHA = ""
)

var bubbleGumPink = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

type Builtin struct {
}

func (c *Builtin) ProvideExecHandler(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		for _, cmd := range []string{"choose", "confirm", "file", "format", "input", "spin", "table", "write"} {
			if args[0] == cmd {
				if err := Run(args[:]); err != nil {
					return err
				}
				return nil
			}
		}
		return next(ctx, args)
	}
}

func Run(args []string) error {
	lipgloss.SetColorProfile(termenv.NewOutput(os.Stderr).Profile)

	if Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
			Version = info.Main.Version
		} else {
			Version = "unknown (built from source)"
		}
	}
	version := fmt.Sprintf("gum version %s", Version)

	gum := &Gum{}
	parser, err := kong.New(
		gum,
		kong.Description(fmt.Sprintf("A tool for %s shell scripts.", bubbleGumPink.Render("glamorous"))),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact:             true,
			Summary:             false,
			NoExpandSubcommands: true,
		}),
		kong.Vars{
			"version":                 version,
			"defaultHeight":           "0",
			"defaultWidth":            "0",
			"defaultAlign":            "left",
			"defaultBorder":           "none",
			"defaultBorderForeground": "",
			"defaultBorderBackground": "",
			"defaultBackground":       "",
			"defaultForeground":       "",
			"defaultMargin":           "0 0",
			"defaultPadding":          "0 0",
			"defaultUnderline":        "false",
			"defaultBold":             "false",
			"defaultFaint":            "false",
			"defaultItalic":           "false",
			"defaultStrikethrough":    "false",
		},
	)
	if err != nil {
		return err
	}
	ctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	return ctx.Run()
}

type Gum struct {
	// Version is a flag that can be used to display the version number.
	Version kong.VersionFlag `short:"v" help:"Print the version number"`

	// Completion generates Gum shell completion scripts.
	Completion completion.Completion `cmd:"" hidden:"" help:"Request shell completion"`

	// Man is a hidden command that generates Gum man pages.
	Man man.Man `cmd:"" hidden:"" help:"Generate man pages"`

	// Choose provides an interface to choose one option from a given list of
	// options. The options can be provided as (new-line separated) stdin or a
	// list of arguments.
	//
	// It is different from the filter command as it does not provide a fuzzy
	// finding input, so it is best used for smaller lists of options.
	//
	// Let's pick from a list of gum flavors:
	//
	// $ gum choose "Strawberry" "Banana" "Cherry"
	//
	Choose choose.Options `cmd:"" help:"Choose an option from a list of choices"`

	// Confirm provides an interface to ask a user to confirm an action.
	// The user is provided with an interface to choose an affirmative or
	// negative answer, which is then reflected in the exit code for use in
	// scripting.
	//
	// If the user selects the affirmative answer, the program exits with 0.
	// If the user selects the negative answer, the program exits with 1.
	//
	// I.e. confirm if the user wants to delete a file
	//
	// $ gum confirm "Are you sure?" && rm file.txt
	//
	Confirm confirm.Options `cmd:"" help:"Ask a user to confirm an action"`

	// File provides an interface to pick a file from a folder (tree).
	// The user is provided a file manager-like interface to navigate, to
	// select a file.
	//
	// Let's pick a file from the current directory:
	//
	// $ gum file
	// $ gum file .
	//
	// Let's pick a file from the home directory:
	//
	// $ gum file $HOME
	File file.Options `cmd:"" help:"Pick a file from a folder"`

	// Filter provides a fuzzy searching text input to allow filtering a list of
	// options to select one option.
	//
	// By default it will list all the files (recursively) in the current directory
	// for the user to choose one, but the script (or user) can provide different
	// new-line separated options to choose from.
	//
	// I.e. let's pick from a list of gum flavors:
	//
	// $ cat flavors.text | gum filter
	//
	// Filter filter.Options `cmd:"" help:"Filter items from a list"`

	// Format allows you to render styled text from `markdown`, `code`,
	// `template` strings, or embedded `emoji` strings.
	// For more information see the format/README.md file.
	Format format.Options `cmd:"" help:"Format a string using a template"`

	// Input provides a shell script interface for the text input bubble.
	// https://github.com/charmbracelet/bubbles/tree/master/textinput
	//
	// It can be used to prompt the user for some input. The text the user
	// entered will be sent to stdout.
	//
	// $ gum input --placeholder "What's your favorite gum?" > answer.text
	//
	Input input.Options `cmd:"" help:"Prompt for some input"`

	// Join provides a shell script interface for the lipgloss JoinHorizontal
	// and JoinVertical commands. It allows you to join multi-line text to
	// build different layouts.
	//
	// For example, you can place two bordered boxes next to each other:
	// Note: We wrap the variable in quotes to ensure the new lines are part of a
	// single argument. Otherwise, the command won't work as expected.
	//
	// $ gum join --horizontal "$BUBBLE_BOX" "$GUM_BOX"
	//
	//   ╔══════════════════════╗╔═════════════╗
	//   ║                      ║║             ║
	//   ║        Bubble        ║║     Gum     ║
	//   ║                      ║║             ║
	//   ╚══════════════════════╝╚═════════════╝
	//
	Join join.Options `cmd:"" help:"Join text vertically or horizontally"`

	// Pager provides a shell script interface for the viewport bubble.
	// https://github.com/charmbracelet/bubbles/tree/master/viewport
	//
	// It allows the user to scroll through content like a pager.
	//
	// ╭────────────────────────────────────────────────╮
	// │    1 │ Gum Pager                               │
	// │    2 │ =========                               │
	// │    3 │                                         │
	// │    4 │ ```                                     │
	// │    5 │ gum pager --height 10 --width 25 < text │
	// │    6 │ ```                                     │
	// │    7 │                                         │
	// │    8 │                                         │
	// ╰────────────────────────────────────────────────╯
	//  ↑/↓: Navigate • q: Quit
	//
	Pager pager.Options `cmd:"" help:"Scroll through a file"`

	// Spin provides a shell script interface for the spinner bubble.
	// https://github.com/charmbracelet/bubbles/tree/master/spinner
	//
	// It is useful for displaying that some task is running in the background
	// while consuming it's output so that it is not shown to the user.
	//
	// For example, let's do a long running task: $ sleep 5
	//
	// We can simply prepend a spinner to this task to show it to the user,
	// while performing the task / command in the background.
	//
	// $ gum spin -t "Taking a nap..." -- sleep 5
	//
	// The spinner will automatically exit when the task is complete.
	//
	Spin spin.Options `cmd:"" help:"Display spinner while running a command"`

	// Style provides a shell script interface for Lip Gloss.
	// https://github.com/charmbracelet/lipgloss
	//
	// It allows you to use Lip Gloss to style text without needing to use Go.
	// All of the styling options are available as flags.
	//
	// Let's make some text glamorous using bash:
	//
	// $ gum style \
	//  	--foreground 212 --border double --align center \
	//  	--width 50 --margin 2 --padding "2 4" \
	//  	"Bubble Gum (1¢)" "So sweet and so fresh\!"
	//
	//
	//    ╔══════════════════════════════════════════════════╗
	//    ║                                                  ║
	//    ║                                                  ║
	//    ║                 Bubble Gum (1¢)                  ║
	//    ║              So sweet and so fresh!              ║
	//    ║                                                  ║
	//    ║                                                  ║
	//    ╚══════════════════════════════════════════════════╝
	//
	Style style.Options `cmd:"" help:"Apply coloring, borders, spacing to text"`

	// Table provides a shell script interface for the table bubble.
	// https://github.com/charmbracelet/bubbles/tree/master/table
	//
	// It is useful to render tabular (CSV) data in a terminal and allows
	// the user to select a row from the table.
	//
	// Let's render a table of gum flavors:
	//
	// $ gum table <<< "Flavor,Price\nStrawberry,$0.50\nBanana,$0.99\nCherry,$0.75"
	//
	//  Flavor      Price
	//  Strawberry  $0.50
	//  Banana      $0.99
	//  Cherry      $0.75
	//
	Table table.Options `cmd:"" help:"Render a table of data"`

	// Write provides a shell script interface for the text area bubble.
	// https://github.com/charmbracelet/bubbles/tree/master/textarea
	//
	// It can be used to ask the user to write some long form of text
	// (multi-line) input. The text the user entered will be sent to stdout.
	//
	// $ gum write > output.text
	//
	Write write.Options `cmd:"" help:"Prompt for long-form text"`
}