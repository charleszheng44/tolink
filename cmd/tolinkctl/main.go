package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/charleszheng44/tolink/pkg/client"
)

const (
	exitSuccess       = 0
	exitFailure       = 1
	exitNotFound      = 2
	exitAlreadyExists = 3
	exitUsage         = 64
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tolinkctl", flag.ContinueOnError)
	fs.SetOutput(stderr)
	defaultData := filepath.Join(os.Getenv("HOME"), ".config", "tolink", "links.json")
	urlFlag := fs.String("url", "http://127.0.0.1:4080", "Daemon base URL")
	dataFlag := fs.String("data", defaultData, "Path to links JSON file")

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	urlExplicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "url" {
			urlExplicit = true
		}
	})

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(stderr, "tolinkctl: missing subcommand")
		return exitUsage
	}

	c, err := client.New(*urlFlag, *dataFlag, urlExplicit)
	if err != nil {
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitFailure
	}

	return dispatch(c, remaining, stdout, stderr)
}

func dispatch(c client.Client, args []string, stdout, stderr io.Writer) int {
	switch args[0] {
	case "list":
		return runList(c, args[1:], stdout, stderr)
	case "get":
		return runGet(c, args[1:], stdout, stderr)
	case "add":
		return runAdd(c, args[1:], stdout, stderr)
	case "update":
		return runUpdate(c, args[1:], stdout, stderr)
	case "delete":
		return runDelete(c, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "tolinkctl: unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

func runList(c client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "tolinkctl: list takes no arguments")
		return exitUsage
	}
	links, err := c.List()
	if err != nil {
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitFailure
	}
	if len(links) == 0 {
		return exitSuccess
	}
	shortcuts := make([]string, 0, len(links))
	for k := range links {
		shortcuts = append(shortcuts, k)
	}
	sort.Strings(shortcuts)
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	for _, s := range shortcuts {
		fmt.Fprintf(tw, "%s\t%s\n", s, links[s])
	}
	tw.Flush()
	return exitSuccess
}

func runGet(c client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "tolinkctl: get requires exactly one argument: <shortcut>")
		return exitUsage
	}
	u, err := c.Get(args[0])
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			fmt.Fprintf(stderr, "tolinkctl: shortcut %q not found\n", args[0])
			return exitNotFound
		}
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitFailure
	}
	fmt.Fprintln(stdout, u)
	return exitSuccess
}

func runAdd(c client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "tolinkctl: add requires exactly two arguments: <shortcut> <url>")
		return exitUsage
	}
	shortcut, rawURL := args[0], args[1]
	if err := validateURL(rawURL); err != nil {
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitUsage
	}
	_, err := c.Get(shortcut)
	if err == nil {
		fmt.Fprintf(stderr, "tolinkctl: shortcut %q already exists\n", shortcut)
		return exitAlreadyExists
	}
	if !errors.Is(err, client.ErrNotFound) {
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitFailure
	}
	if err := c.Set(shortcut, rawURL); err != nil {
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitFailure
	}
	return exitSuccess
}

func runUpdate(c client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "tolinkctl: update requires exactly two arguments: <shortcut> <url>")
		return exitUsage
	}
	shortcut, rawURL := args[0], args[1]
	if err := validateURL(rawURL); err != nil {
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitUsage
	}
	_, err := c.Get(shortcut)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			fmt.Fprintf(stderr, "tolinkctl: shortcut %q not found\n", shortcut)
			return exitNotFound
		}
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitFailure
	}
	if err := c.Set(shortcut, rawURL); err != nil {
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitFailure
	}
	return exitSuccess
}

func runDelete(c client.Client, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "tolinkctl: delete requires exactly one argument: <shortcut>")
		return exitUsage
	}
	err := c.Delete(args[0])
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			fmt.Fprintf(stderr, "tolinkctl: shortcut %q not found\n", args[0])
			return exitNotFound
		}
		fmt.Fprintf(stderr, "tolinkctl: %v\n", err)
		return exitFailure
	}
	return exitSuccess
}

func validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url must start with http:// or https://")
	}
	return nil
}
