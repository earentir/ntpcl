package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/beevik/ntp"
	cli "github.com/jawher/mow.cli"
)

// setSystemTime sets the system time to the specified time.
func setSystemTime(t time.Time) error {
	var cmd *exec.Cmd
	formattedTime := t.Format("2006-01-02 15:04:05.000000000")

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "date", formattedTime[:10])
		if err := cmd.Run(); err != nil {
			return err
		}
		cmd = exec.Command("cmd", "/C", "time", formattedTime[11:])
	case "linux":
		cmd = exec.Command("sudo", "date", "-s", formattedTime)
	case "darwin":
		cmd = exec.Command("sudo", "date", "-u", formattedTime)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Run()
}

func getServerIP(server string) (string, error) {
	ips, err := net.LookupIP(server)
	if err != nil {
		return "", err
	}
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 address found for server %s", server)
}

func queryNTPTime(server string) (*ntp.Response, time.Duration, error) {
	start := time.Now()
	response, err := ntp.Query(server)
	if err != nil {
		return nil, 0, err
	}
	rtt := time.Since(start)
	return response, rtt, nil
}

func getTimeFromHTTPHeader(url string) (time.Time, time.Duration, int, error) {
	start := time.Now()
	resp, err := http.Head(url)
	if err != nil {
		return time.Time{}, 0, 0, err
	}
	rtt := time.Since(start)
	defer resp.Body.Close()

	dateHeader := resp.Header.Get("Date")
	if dateHeader == "" {
		return time.Time{}, 0, 0, fmt.Errorf("no Date header found in response")
	}

	serverTime, err := time.Parse(time.RFC1123, dateHeader)
	if err != nil {
		return time.Time{}, 0, 0, err
	}

	return serverTime, rtt, resp.StatusCode, nil
}

func querySNTPTime(server string) (*ntp.Response, time.Duration, error) {
	return queryNTPTime(server)
}

func main() {
	app := cli.App("timeclient", "A simple time client to fetch and optionally set system time")
	app.LongDesc = "A simple NTP client to fetch and optionally set system time. It can be used to query an NTP server for the current time and set the system time to the retrieved time.\nhttps://github.com/earentir/ntpcl"
	app.Version("v version", "0.3.7")

	var (
		ntpServer         = app.StringOpt("ntp-server n", "europe.pool.ntp.org", "NTP server to query")
		httpURL           = app.StringOpt("url u", "", "URL to query for time from HTTP header")
		setTime           = app.BoolOpt("set s", false, "Set the system time")
		highAccuracy      = app.BoolOpt("high-accuracy h", false, "Use high accuracy mode")
		windowsTimeServer = app.StringOpt("windows-time-server w", "", "Windows Time Server to query")
	)

	app.Action = func() {
		// Disallow using both HTTP URL and high accuracy mode
		if *httpURL != "" && *highAccuracy {
			log.Fatalf("Cannot use --url and --high-accuracy together")
		}

		var serverTime time.Time
		var roundTripTime time.Duration
		var err error

		if *httpURL != "" {
			fmt.Printf("Querying time from HTTP server: %s\n", *httpURL)
			var statusCode int
			serverTime, roundTripTime, statusCode, err = getTimeFromHTTPHeader(*httpURL)
			if err != nil {
				log.Fatalf("Failed to get time from HTTP server: %v", err)
			}
			fmt.Printf("Status Code: %d\n", statusCode)
			fmt.Printf("WEB   : %v\n", serverTime)
			fmt.Printf("RTT   : %v\n", roundTripTime)
		} else {
			var ntpServerToUse string
			var response *ntp.Response
			var rtt time.Duration
			var queryMethod string

			if *windowsTimeServer != "" {
				ntpServerToUse = *windowsTimeServer
			} else {
				ntpServerToUse = *ntpServer
			}

			serverIP, err := getServerIP(ntpServerToUse)
			if err != nil {
				log.Fatalf("Failed to get IP address for server: %v", err)
			}

			response, rtt, err = queryNTPTime(ntpServerToUse)
			queryMethod = "NTP"
			if err != nil {
				response, rtt, err = querySNTPTime(ntpServerToUse)
				queryMethod = "SNTP"
				if err != nil {
					log.Fatalf("Failed to query server using NTP and SNTP: %v", err)
				}
			}

			serverTime = time.Now().Add(response.ClockOffset)

			// Print the NTP/SNTP response details
			fmt.Printf("%s   : %v\n", queryMethod, serverTime)
			fmt.Printf("Local : %v\n", time.Now())
			fmt.Printf("RTT   : %v\n", rtt)
			fmt.Printf("Server: %s (%s)\n", ntpServerToUse, serverIP)
			fmt.Printf("Stratum: %d\n", response.Stratum)
			fmt.Printf("Precision: %d\n", response.Precision)
			fmt.Printf("Root Delay: %v\n", response.RootDelay)
			fmt.Printf("Root Dispersion: %v\n", response.RootDispersion)
			fmt.Printf("Round-trip delay: %v\n", response.RTT)
			fmt.Printf("Clock offset: %v\n", response.ClockOffset)
			fmt.Printf("Poll interval: %v\n", response.Poll)

			if *highAccuracy {
				fmt.Println("High accuracy mode enabled. Gathering multiple samples...")
				var offsets []time.Duration
				for i := 0; i < 8; i++ {
					sampleResponse, _, err := queryNTPTime(ntpServerToUse)
					if err != nil {
						log.Fatalf("Failed to query server: %v", err)
					}
					offsets = append(offsets, sampleResponse.ClockOffset)
					time.Sleep(500 * time.Millisecond)
				}

				// Calculate the average offset.
				var totalOffset time.Duration
				for _, offset := range offsets {
					totalOffset += offset
				}
				averageOffset := totalOffset / time.Duration(len(offsets))
				serverTime = time.Now().Add(averageOffset)

				fmt.Printf("Average offset: %v\n", averageOffset)
				fmt.Printf("Adjusted time: %v\n", serverTime)
			}
		}

		localTime := time.Now()
		if *httpURL != "" {
			fmt.Printf("Local : %v\n", localTime)
		} else {
			fmt.Printf("Time  : %v\n", serverTime)
			fmt.Printf("Local : %v\n", localTime)
		}

		fmt.Printf("Time difference: %v\n", serverTime.Sub(localTime))
		if *httpURL != "" {
			fmt.Printf("RTT   : %v\n", roundTripTime)
			fmt.Printf("Retrieved from HTTP server: %s\n", *httpURL)
		}

		// Set the system time if the flag is set.
		if *setTime {
			if err := setSystemTime(serverTime); err != nil {
				log.Fatalf("Failed to set system time: %v", err)
			}
			fmt.Println("System time updated successfully")

			// Verify the new system time.
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
