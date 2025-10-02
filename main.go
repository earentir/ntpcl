package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"ntpcl/timeutils"

	"github.com/beevik/ntp"
	cli "github.com/jawher/mow.cli"
)

func main() {
	app := cli.App("timeclient", "A simple time client to fetch and optionally set system time")
	app.LongDesc = "A simple time client to fetch and optionally set system time. It can be used to query an NTP server, HTTP server, Daytime Protocol server, or Time Protocol server for the current time and set the system time to the retrieved time.\nhttps://github.com/earentir/ntpcl"
	app.Version("v version", "0.4.19")

	var (
		ntpServer          = app.StringOpt("ntp-server", "europe.pool.ntp.org", "NTP server to query")
		httpURL            = app.StringOpt("http-server", "", "URL to query for time from HTTP header")
		daytimeServer      = app.StringOpt("daytime-server", "", "Daytime Protocol server to query")
		timeProtocolServer = app.StringOpt("time-server", "", "Time Protocol server to query")
		windowsTimeServer  = app.StringOpt("windows-time-server", "", "Windows Time Server to query")
		setTime            = app.BoolOpt("set", false, "Set the system time")
		highAccuracy       = app.BoolOpt("high-accuracy", false, "Use high accuracy mode (only with NTP)")
		useSystemTools     = app.BoolOpt("system-tools", false, "Use system commands to set time instead of system calls")
	)

	app.Action = func() {
		sources := []*string{httpURL, daytimeServer, timeProtocolServer, ntpServer, windowsTimeServer}
		if countNonEmptySources(sources) > 1 {
			log.Fatal("Only one time source can be selected.")
		}

		if *highAccuracy && *ntpServer == "" && *windowsTimeServer == "" {
			log.Fatal("--high-accuracy can only be used with NTP.")
		}

		serverTime, roundTripTime, ntpResponse, server, err := fetchTime(httpURL, daytimeServer, timeProtocolServer, ntpServer, windowsTimeServer, *highAccuracy)
		if err != nil {
			log.Fatalf("Failed to fetch time: %v", err)
		}

		method := determineMethod(httpURL, daytimeServer, timeProtocolServer, ntpServer, windowsTimeServer)
		timeutils.DisplayTimeInfo(method, serverTime, roundTripTime, server, ntpResponse)

		if *setTime {
			if err := timeutils.SetSystemTimeWrapper(serverTime, *useSystemTools); err != nil {
				log.Fatalf("Failed to set system time: %v", err)
			}
			fmt.Println("System time updated successfully")
			printNewTimeInfo(serverTime)
		}
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Failed to run the app: %v", err)
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
