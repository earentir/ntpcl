package main

import (
	"fmt"
	"log"
	"net"
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

func queryNTPTime(server string) (*ntp.Response, error) {
	response, err := ntp.Query(server)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func main() {
	app := cli.App("ntpclient", "A simple NTP client to fetch and optionally set system time")
	app.LongDesc = "A simple NTP client to fetch and optionally set system time. It can be used to query an NTP server for the current time and set the system time to the retrieved time.\nhttps://github.com/earentir/ntpcl"
	app.Version("v version", "0.1.1")

	var (
		ntpServer    = app.StringOpt("server s", "europe.pool.ntp.org", "NTP server to query")
		setTime      = app.BoolOpt("set t", false, "Set the system time")
		highAccuracy = app.BoolOpt("high-accuracy a", false, "Use high accuracy mode")
	)

	app.Action = func() {
		// Get the IP address of the NTP server.
		serverIP, err := getServerIP(*ntpServer)
		if err != nil {
			log.Fatalf("Failed to get IP address for NTP server: %v", err)
		}

		// Query the NTP server for the current time.
		response, err := queryNTPTime(*ntpServer)
		if err != nil {
			log.Fatalf("Failed to query NTP server: %v", err)
		}

		ntpTime := time.Now().Add(response.ClockOffset)
		localTime := time.Now()

		// Print the times and retrieval details.
		fmt.Printf("NTP   : %v\n", ntpTime)
		fmt.Printf("Local : %v\n", localTime)
		fmt.Printf("NTP server: %s (%s)\n", *ntpServer, serverIP)
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
				sampleResponse, err := queryNTPTime(*ntpServer)
				if err != nil {
					log.Fatalf("Failed to query NTP server: %v", err)
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
			ntpTime = time.Now().Add(averageOffset)

			fmt.Printf("Average offset: %v\n", averageOffset)
			fmt.Printf("Adjusted NTP time: %v\n", ntpTime)
		}

		// Set the system time if the flag is set.
		if *setTime {
			if err := setSystemTime(ntpTime); err != nil {
				log.Fatalf("Failed to set system time: %v", err)
			}
			fmt.Println("System time updated successfully")

			// Verify the new system time.
			newLocalTime := time.Now()
			fmt.Printf("New local system time: %v\n", newLocalTime)
			fmt.Printf("Time difference after setting: %v\n", newLocalTime.Sub(ntpTime))
		}
	}

	app.Run(os.Args)
}
