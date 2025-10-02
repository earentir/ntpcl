package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"ntpcl/timeutils"

	"github.com/beevik/ntp"
	"github.com/spf13/cobra"
)

const defaultNTPServer = "europe.pool.ntp.org"

type commandOptions struct {
	set            bool
	useSystemTools bool
	highAccuracy   bool
}

func main() {
	var rootOpts commandOptions

	rootCmd := &cobra.Command{
		Use:     "timeclient",
		Short:   "A simple time client to fetch and optionally set system time",
		Long:    "A simple time client to fetch and optionally set system time. It can be used to query an NTP server, HTTP server, Daytime Protocol server, or Time Protocol server for the current time and set the system time to the retrieved time.\nhttps://github.com/earentir/ntpcl",
		Version: "0.4.19",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithSource(rootOpts, "", "", "", defaultNTPServer, "")
		},
	}

	addCommonFlags(rootCmd, &rootOpts)
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.SetVersionTemplate("{{printf \"%s\" .Version}}\n")

	var ntpOpts commandOptions
	ntpCmd := &cobra.Command{
		Use:   "ntp SERVER",
		Short: "Fetch time from an NTP server",
		Args:  requireSingleArg("ntp", "an NTP server address"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithSource(ntpOpts, "", "", "", args[0], "")
		},
	}
	addCommonFlags(ntpCmd, &ntpOpts)

	var httpOpts commandOptions
	httpCmd := &cobra.Command{
		Use:   "http URL",
		Short: "Fetch time from an HTTP server's Date header",
		Args:  requireSingleArg("http", "a URL"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithSource(httpOpts, args[0], "", "", "", "")
		},
	}
	addCommonFlags(httpCmd, &httpOpts)

	var daytimeOpts commandOptions
	daytimeCmd := &cobra.Command{
		Use:   "daytime SERVER",
		Short: "Fetch time from a Daytime Protocol server",
		Args:  requireSingleArg("daytime", "a server address"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithSource(daytimeOpts, "", args[0], "", "", "")
		},
	}
	addCommonFlags(daytimeCmd, &daytimeOpts)

	var timeOpts commandOptions
	timeCmd := &cobra.Command{
		Use:   "time SERVER",
		Short: "Fetch time from a Time Protocol (RFC 868) server",
		Args:  requireSingleArg("time", "a server address"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithSource(timeOpts, "", "", args[0], "", "")
		},
	}
	addCommonFlags(timeCmd, &timeOpts)

	var windowsOpts commandOptions
	windowsCmd := &cobra.Command{
		Use:   "windows-time SERVER",
		Short: "Fetch time from a Windows Time server",
		Args:  requireSingleArg("windows-time", "a server address"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithSource(windowsOpts, "", "", "", "", args[0])
		},
	}
	addCommonFlags(windowsCmd, &windowsOpts)

	rootCmd.AddCommand(ntpCmd, httpCmd, daytimeCmd, timeCmd, windowsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addCommonFlags(cmd *cobra.Command, opts *commandOptions) {
	cmd.Flags().BoolVar(&opts.set, "set", false, "Set the system time")
	cmd.Flags().BoolVar(&opts.useSystemTools, "system-tools", false, "Use system commands to set time instead of system calls")
	cmd.Flags().BoolVar(&opts.highAccuracy, "high-accuracy", false, "Use high accuracy mode (only with NTP)")
}

func requireSingleArg(commandName, argDescription string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errorWithCommandHelp(cmd, fmt.Sprintf("missing %s: provide %s when using the %s command", argDescription, argDescription, commandName))
		}
		if len(args) > 1 {
			return errorWithCommandHelp(cmd, fmt.Sprintf("the %s command accepts only one argument (%s)", commandName, argDescription))
		}
		return nil
	}
}

func errorWithCommandHelp(cmd *cobra.Command, message string) error {
	var help bytes.Buffer
	originalOut := cmd.OutOrStdout()
	originalErr := cmd.ErrOrStderr()
	cmd.SetOut(&help)
	cmd.SetErr(&help)
	_ = cmd.Help()
	cmd.SetOut(originalOut)
	cmd.SetErr(originalErr)
	return fmt.Errorf("%s\n\n%s", message, help.String())
}

func runWithSource(opts commandOptions, httpURL, daytimeServer, timeProtocolServer, ntpServer, windowsTimeServer string) error {
	sources := []*string{&httpURL, &daytimeServer, &timeProtocolServer, &ntpServer, &windowsTimeServer}
	if countNonEmptySources(sources) > 1 {
		return fmt.Errorf("Only one time source can be selected.")
	}

	if opts.highAccuracy && ntpServer == "" && windowsTimeServer == "" {
		return fmt.Errorf("--high-accuracy can only be used with NTP.")
	}

	serverTime, roundTripTime, ntpResponse, server, err := fetchTime(&httpURL, &daytimeServer, &timeProtocolServer, &ntpServer, &windowsTimeServer, opts.highAccuracy)
	if err != nil {
		return fmt.Errorf("Failed to fetch time: %w", err)
	}

	method := determineMethod(&httpURL, &daytimeServer, &timeProtocolServer, &ntpServer, &windowsTimeServer)
	timeutils.DisplayTimeInfo(method, serverTime, roundTripTime, server, ntpResponse)

	if opts.set {
		if err := timeutils.SetSystemTimeWrapper(serverTime, opts.useSystemTools); err != nil {
			return fmt.Errorf("Failed to set system time: %w", err)
		}
		fmt.Println("System time updated successfully")
		printNewTimeInfo(serverTime)
	}

	return nil
}

func countNonEmptySources(sources []*string) int {
	count := 0
	for _, source := range sources {
		if *source != "" {
			count++
		}
	}
	return count
}

func fetchTime(httpURL, daytimeServer, timeProtocolServer, ntpServer, windowsTimeServer *string, highAccuracy bool) (time.Time, time.Duration, *ntp.Response, string, error) {
	switch {
	case *httpURL != "":
		t, rtt, err := timeutils.FetchTimeFromHTTP(*httpURL)
		return t, rtt, nil, *httpURL, err
	case *daytimeServer != "":
		t, rtt, err := timeutils.FetchTimeFromDaytimeProtocol(*daytimeServer)
		return t, rtt, nil, *daytimeServer, err
	case *timeProtocolServer != "":
		t, rtt, err := timeutils.FetchTimeFromTimeProtocol(*timeProtocolServer)
		return t, rtt, nil, *timeProtocolServer, err
	case *ntpServer != "":
		return timeutils.FetchTimeFromNTP(*ntpServer, "", highAccuracy)
	case *windowsTimeServer != "":
		return timeutils.FetchTimeFromNTP("", *windowsTimeServer, highAccuracy)
	default:
		return timeutils.FetchTimeFromNTP("europe.pool.ntp.org", "", highAccuracy)
	}
}

func determineMethod(httpURL, daytimeServer, timeProtocolServer, ntpServer, windowsTimeServer *string) string {
	switch {
	case *httpURL != "":
		return "HTTP"
	case *daytimeServer != "":
		return "Daytime"
	case *timeProtocolServer != "":
		return "Time Protocol"
	case *ntpServer != "", *windowsTimeServer != "":
		return "NTP"
	default:
		return "NTP"
	}
}

func printNewTimeInfo(serverTime time.Time) {
	newLocalTime := time.Now()
	timeDiff := newLocalTime.Sub(serverTime)
	fmt.Print(timeutils.FormattedOutput("Local Time Update", newLocalTime, serverTime, timeDiff, 0, "", nil))
}
