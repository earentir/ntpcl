package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/beevik/ntp"
)

// fetchTimeFromDaytimeProtocol fetches the time from a server using the Daytime Protocol (RFC 867).
func fetchTimeFromDaytimeProtocol(server string) (time.Time, time.Duration, error) {
	start := time.Now()
	conn, err := net.Dial("tcp", net.JoinHostPort(server, "13"))
	if err != nil {
		return time.Time{}, 0, err
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		return time.Time{}, 0, err
	}

	rtt := time.Since(start)

	// Debug: Print raw Daytime response
	fmt.Printf("Raw Daytime response: %s\n", response)

	serverTime, err := parseDaytimeResponse(response)
	if err != nil {
		return time.Time{}, 0, err
	}

	return serverTime, rtt, nil
}

// parseDaytimeResponse parses the response from the Daytime Protocol to extract the time.
func parseDaytimeResponse(response string) (time.Time, error) {
	// Example of typical daytime response: "Tue Jun 27 19:32:37 2024\n"
	response = strings.TrimSpace(response)
	serverTime, err := time.Parse("Mon Jan 2 15:04:05 2006", response)
	if err != nil {
		return time.Time{}, err
	}
	return serverTime, nil
}

// fetchTimeFromTimeProtocol fetches the time from a server using the Time Protocol (RFC 868).
func fetchTimeFromTimeProtocol(server string) (time.Time, time.Duration, error) {
	start := time.Now()
	conn, err := net.Dial("udp", net.JoinHostPort(server, "37"))
	if err != nil {
		return time.Time{}, 0, err
	}
	defer conn.Close()

	buffer := make([]byte, 4)
	n, err := conn.Read(buffer)
	if err != nil {
		return time.Time{}, 0, err
	}
	if n != 4 {
		return time.Time{}, 0, fmt.Errorf("invalid response size")
	}

	rtt := time.Since(start)

	seconds := binary.BigEndian.Uint32(buffer)
	// The Time Protocol returns the number of seconds since 00:00 (midnight) 1 January 1900 GMT
	unixTime := int64(seconds) - 2208988800 // Convert to Unix time (seconds since 1970)

	serverTime := time.Unix(unixTime, 0).UTC()

	return serverTime, rtt, nil
}

// fetchTimeFromHTTP fetches the time from an HTTP server's Date header.
func fetchTimeFromHTTP(url string) (time.Time, time.Duration, error) {
	start := time.Now()
	resp, err := http.Head(url)
	if err != nil {
		return time.Time{}, 0, err
	}
	rtt := time.Since(start)
	defer resp.Body.Close()

	dateHeader := resp.Header.Get("Date")
	if dateHeader == "" {
		return time.Time{}, 0, fmt.Errorf("no Date header found in response")
	}

	serverTime, err := time.Parse(time.RFC1123, dateHeader)
	if err != nil {
		return time.Time{}, 0, err
	}

	return serverTime, rtt, nil
}

// fetchTimeFromNTP fetches the time from an NTP server.
func fetchTimeFromNTP(ntpServer, windowsTimeServer string, highAccuracy bool) (time.Time, time.Duration, error) {
	var serverTime time.Time
	var rtt time.Duration
	var err error

	var ntpServerToUse string
	if windowsTimeServer != "" {
		ntpServerToUse = windowsTimeServer
	} else {
		ntpServerToUse = ntpServer
	}

	// Check if ntpServerToUse is an IP address
	if net.ParseIP(ntpServerToUse) == nil {
		// If it's not an IP address, resolve the hostname
		ntpServerToUse, err = getServerIP(ntpServerToUse)
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("failed to get IP address for server: %v", err)
		}
	}

	response, rtt, err := queryNTPTime(ntpServerToUse)
	queryMethod := "NTP"
	if err != nil {
		response, rtt, err = querySNTPTime(ntpServerToUse)
		queryMethod = "SNTP"
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("failed to query server using NTP and SNTP: %v", err)
		}
	}

	serverTime = time.Now().Add(response.ClockOffset)
	printNTPDetails(queryMethod, serverTime, rtt, ntpServerToUse, ntpServerToUse, response)

	if highAccuracy {
		serverTime, err = gatherHighAccuracyTime(ntpServerToUse)
		if err != nil {
			return time.Time{}, 0, err
		}
	}

	return serverTime, rtt, nil
}

// gatherHighAccuracyTime gathers multiple samples to get a high accuracy time.
func gatherHighAccuracyTime(ntpServerToUse string) (time.Time, error) {
	fmt.Println("High accuracy mode enabled. Gathering multiple samples...")
	var offsets []time.Duration
	for i := 0; i < 8; i++ {
		sampleResponse, _, err := queryNTPTime(ntpServerToUse)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to query server: %v", err)
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
	serverTime := time.Now().Add(averageOffset)

	fmt.Printf("Average offset: %v\n", averageOffset)
	fmt.Printf("Adjusted time: %v\n", serverTime)
	return serverTime, nil
}

// setSystemTimeWrapper decides whether to use system calls or system commands.
func setSystemTimeWrapper(t time.Time, useSystemTools bool) error {
	if useSystemTools {
		return setSystemTimeWithCommand(t)
	}
	return setSystemTime(t)
}

// setSystemTimeWithCommand sets the system time using system commands.
func setSystemTimeWithCommand(t time.Time) error {
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

// getServerIP resolves the IP address of the server.
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

// queryNTPTime queries the NTP server for the current time.
func queryNTPTime(server string) (*ntp.Response, time.Duration, error) {
	start := time.Now()
	response, err := ntp.Query(server)
	if err != nil {
		return nil, 0, err
	}
	rtt := time.Since(start)
	return response, rtt, nil
}

// querySNTPTime queries the SNTP server for the current time.
func querySNTPTime(server string) (*ntp.Response, time.Duration, error) {
	return queryNTPTime(server)
}

// displayTimeInfo displays the fetched time and round trip time.
func displayTimeInfo(serverTime time.Time, roundTripTime time.Duration, httpURL string) {
	localTime := time.Now()
	if httpURL != "" {
		fmt.Printf("WEB   : %v\n", serverTime)
		fmt.Printf("Local : %v\n", localTime)
	} else {
		fmt.Printf("Time  : %v\n", serverTime)
		fmt.Printf("Local : %v\n", localTime)
	}
	fmt.Printf("Time difference: %v\n", serverTime.Sub(localTime))
	fmt.Printf("RTT   : %v\n", roundTripTime)
	if httpURL != "" {
		fmt.Printf("Retrieved from HTTP server: %s\n", httpURL)
	}
}

// printNTPDetails prints the details of the NTP response.
func printNTPDetails(method string, serverTime time.Time, rtt time.Duration, server, serverIP string, response *ntp.Response) {
	fmt.Printf("%s   : %v\n", method, serverTime)
	fmt.Printf("Local : %v\n", time.Now())
	fmt.Printf("RTT   : %v\n", rtt)
	fmt.Printf("Server: %s (%s)\n", server, serverIP)
	fmt.Printf("Stratum: %d\n", response.Stratum)
	fmt.Printf("Precision: %d\n", response.Precision)
	fmt.Printf("Root Delay: %v\n", response.RootDelay)
	fmt.Printf("Root Dispersion: %v\n", response.RootDispersion)
	fmt.Printf("Round-trip delay: %v\n", response.RTT)
	fmt.Printf("Clock offset: %v\n", response.ClockOffset)
	fmt.Printf("Poll interval: %v\n", response.Poll)
}
