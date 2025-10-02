package main

import (
	"fmt"
	"log"
	"time"

	"ntpcl/timeutils"

	"github.com/beevik/ntp"
	"github.com/spf13/cobra"
)

func main() {
	var (
		ntpServer          string
		httpURL            string
		daytimeServer      string
		timeProtocolServer string
		windowsTimeServer  string
		setTime            bool
		highAccuracy       bool
		useSystemTools     bool
	)

	rootCmd := &cobra.Command{
		Use:     "timeclient",
		Short:   "A simple time client to fetch and optionally set system time",
		Long:    "A simple time client to fetch and optionally set system time. It can be used to query an NTP server, HTTP server, Daytime Protocol server, or Time Protocol server for the current time and set the system time to the retrieved time.\nhttps://github.com/earentir/ntpcl",
		Version: "0.4.19",
		RunE: func(cmd *cobra.Command, args []string) error {
			sources := []*string{&httpURL, &daytimeServer, &timeProtocolServer, &ntpServer, &windowsTimeServer}
			if countNonEmptySources(sources) > 1 {
				return fmt.Errorf("Only one time source can be selected.")
			}

			if highAccuracy && ntpServer == "" && windowsTimeServer == "" {
				return fmt.Errorf("--high-accuracy can only be used with NTP.")
			}

			serverTime, roundTripTime, ntpResponse, server, err := fetchTime(&httpURL, &daytimeServer, &timeProtocolServer, &ntpServer, &windowsTimeServer, highAccuracy)
			if err != nil {
				return fmt.Errorf("Failed to fetch time: %w", err)
			}

			method := determineMethod(&httpURL, &daytimeServer, &timeProtocolServer, &ntpServer, &windowsTimeServer)
			timeutils.DisplayTimeInfo(method, serverTime, roundTripTime, server, ntpResponse)

			if setTime {
				if err := timeutils.SetSystemTimeWrapper(serverTime, useSystemTools); err != nil {
					return fmt.Errorf("Failed to set system time: %w", err)
				}
				fmt.Println("System time updated successfully")
				printNewTimeInfo(serverTime)
			}

			return nil
		},
	}

	rootCmd.Flags().StringVar(&ntpServer, "ntp-server", "europe.pool.ntp.org", "NTP server to query")
	rootCmd.Flags().StringVar(&httpURL, "http-server", "", "URL to query for time from HTTP header")
	rootCmd.Flags().StringVar(&daytimeServer, "daytime-server", "", "Daytime Protocol server to query")
	rootCmd.Flags().StringVar(&timeProtocolServer, "time-server", "", "Time Protocol server to query")
	rootCmd.Flags().StringVar(&windowsTimeServer, "windows-time-server", "", "Windows Time Server to query")
	rootCmd.Flags().BoolVar(&setTime, "set", false, "Set the system time")
	rootCmd.Flags().BoolVar(&highAccuracy, "high-accuracy", false, "Use high accuracy mode (only with NTP)")
	rootCmd.Flags().BoolVar(&useSystemTools, "system-tools", false, "Use system commands to set time instead of system calls")
	rootCmd.SilenceUsage = true
	rootCmd.SetVersionTemplate("{{printf \"%s\" .Version}}\n")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("%v", err)
	}
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
