package main

import (
	"fmt"
	"log"
	"os"
	"time"

	cli "github.com/jawher/mow.cli"
)

func main() {
	app := cli.App("timeclient", "A simple time client to fetch and optionally set system time")
	app.LongDesc = "A simple time client to fetch and optionally set system time. It can be used to query an NTP server, HTTP server, Daytime Protocol server, or Time Protocol server for the current time and set the system time to the retrieved time.\nhttps://github.com/earentir/ntpcl"
	app.Version("v version", "0.4.13")

	var (
		ntpServer          = app.StringOpt("ntp-server n", "europe.pool.ntp.org", "NTP server to query")
		httpURL            = app.StringOpt("url u", "", "URL to query for time from HTTP header")
		daytimeServer      = app.StringOpt("daytime-server d", "", "Daytime Protocol server to query")
		timeProtocolServer = app.StringOpt("time-server t", "", "Time Protocol server to query")
		windowsTimeServer  = app.StringOpt("windows-time-server w", "", "Windows Time Server to query")
		setTime            = app.BoolOpt("set s", false, "Set the system time")
		highAccuracy       = app.BoolOpt("high-accuracy h", false, "Use high accuracy mode (only with NTP)")
		useSystemTools     = app.BoolOpt("system-tools c", false, "Use system commands to set time instead of system calls")
	)

	app.Action = func() {
		// Ensure mutually exclusive options
		selectedSources := 0
		if *httpURL != "" {
			selectedSources++
		}
		if *daytimeServer != "" {
			selectedSources++
		}
		if *timeProtocolServer != "" {
			selectedSources++
		}
		if *ntpServer != "" {
			selectedSources++
		}
		if *windowsTimeServer != "" {
			selectedSources++
		}

		if selectedSources > 1 {
			log.Fatalf("Only one time source can be selected (NTP, HTTP, Daytime Protocol, Time Protocol, or Windows Time Server).")
		}

		if *highAccuracy && *ntpServer == "" && *windowsTimeServer == "" {
			log.Fatalf("--high-accuracy can only be used with NTP.")
		}

		var serverTime time.Time
		var roundTripTime time.Duration
		var err error

		switch {
		case *httpURL != "":
			serverTime, roundTripTime, err = fetchTimeFromHTTP(*httpURL)
		case *daytimeServer != "":
			serverTime, roundTripTime, err = fetchTimeFromDaytimeProtocol(*daytimeServer)
		case *timeProtocolServer != "":
			serverTime, roundTripTime, err = fetchTimeFromTimeProtocol(*timeProtocolServer)
		case *ntpServer != "":
			serverTime, roundTripTime, err = fetchTimeFromNTP(*ntpServer, "", *highAccuracy)
		case *windowsTimeServer != "":
			serverTime, roundTripTime, err = fetchTimeFromNTP(*windowsTimeServer, "", *highAccuracy)
		default:
			// Default to NTP if no source is selected
			serverTime, roundTripTime, err = fetchTimeFromNTP("europe.pool.ntp.org", "", *highAccuracy)
		}

		if err != nil {
			log.Fatalf("Failed to fetch time: %v", err)
		}

		displayTimeInfo(serverTime, roundTripTime, *httpURL)

		if *setTime {
			err = setSystemTimeWrapper(serverTime, *useSystemTools)
			if err != nil {
				log.Fatalf("Failed to set system time: %v", err)
			}
			fmt.Println("System time updated successfully")

			newLocalTime := time.Now()
			fmt.Printf("New local system time: %v\n", newLocalTime)
			fmt.Printf("Time difference after setting: %v\n", newLocalTime.Sub(serverTime))
		}
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("Failed to run the app: %v", err)
	}
}
