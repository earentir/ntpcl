# ntpcl
A simple NTP client to fetch and optionally set system time

## Default Run
By just running the app you get the NTP server time, the local time and many details for the server and the time.

```bash
./ntpcl
```
```bash
NTP   : 2024-06-26 23:43:34.038635381 +0300 EEST m=+0.097589882
Local : 2024-06-26 23:43:34.041431512 +0300 EEST m=+0.100386014
NTP server: europe.pool.ntp.org (80.89.32.122)
Stratum: 2
Precision: 953
Root Delay: 21.896362ms
Root Dispersion: 32.653809ms
Round-trip delay: 82.871417ms
Clock offset: -2.79595ms
Poll interval: 8s
```

## Usage

### Set Time to NTP Server
```bash
./ntpcl -s
```
```bash
NTP   : 2024-06-26 23:44:48.532935783 +0300 EEST m=+0.095169689
Local : 2024-06-26 23:44:48.535802442 +0300 EEST m=+0.098036349
NTP server: europe.pool.ntp.org (131.111.8.63)
Stratum: 2
Precision: 953
Root Delay: 21.896362ms
Root Dispersion: 33.7677ms
Round-trip delay: 84.05751ms
Clock offset: -2.866526ms
Poll interval: 8s
System time updated successfully
New local system time: 2024-06-26 23:44:48.536349515 +0300 EEST m=+0.179418170
Time difference after setting: 84.248481ms
```
### Set Time to Custom NTP Server
```bash
./ntpcl --ntp-server europe.pool.ntp.org --set
```
```bash
NTP   : 2024-06-26 23:45:34.414975883 +0300 EEST m=+0.137681722
Local : 2024-06-26 23:45:34.33176466 +0300 EEST m=+0.054470499
NTP server: europe.pool.ntp.org (158.101.213.248)
Stratum: 2
Precision: 953
Root Delay: 350.952µs
Root Dispersion: 930.786µs
Round-trip delay: 38.742872ms
Clock offset: 83.211354ms
Poll interval: 1s
System time updated successfully
New local system time: 2024-06-26 23:45:34.41829209 +0300 EEST m=+0.107186288
Time difference after setting: -30.495434ms
```

### High Accuracy Mode
```bash
./ntpcl --ntp-server europe.pool.ntp.org --high-accuracy --set
```
```bash
NTP   : 2024-06-26 23:46:37.747730162 +0300 EEST m=+0.116907113
Local : 2024-06-26 23:46:37.681243144 +0300 EEST m=+0.050420097
NTP server: europe.pool.ntp.org (49.12.125.53)
Stratum: 2
Precision: 953
Root Delay: 350.952µs
Root Dispersion: 991.821µs
Round-trip delay: 39.303808ms
Clock offset: 66.487121ms
Poll interval: 1s
High accuracy mode enabled. Gathering multiple samples...
Average offset: 64.595344ms
Adjusted NTP time: 2024-06-26 23:46:42.212707382 +0300 EEST m=+4.581884334
System time updated successfully
New local system time: 2024-06-26 23:46:42.216784924 +0300 EEST m=+4.577605048
Time difference after setting: -4.279286ms
```

### Web Time
```bash
./ntpcl --http-server https://google.com
```
```bash
Querying time from HTTP server: https://google.com
Status Code: 200
WEB   : 2024-06-27 14:53:10 +0000 GMT
RTT   : 1.122863309s
Local : 2024-06-27 17:53:11.276849997 +0300 EEST m=+1.123107442
Time difference: -1.276849997s
RTT   : 1.122863309s
Retrieved from HTTP server: https://google.co
```
