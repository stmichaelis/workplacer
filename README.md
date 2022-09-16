Workplacer switches your Mattermost custom status depending on the IP of your device.

In times where you may be working from home or at the office on an irregular basis, it could be 
beneficial to let your co-workers know in Mattermost where you currently are sitting. This small script sets your custom status based on the IP your device received, assuming that those from home and at work are different.

# Usage

```
  -acidr string
        -acidr <Class Inter-Doman Routing>: CIDR address of network A, e.g. 192.168.1.0/24 for all ip addresses in 192.168.1.*
  -aemoji string
        -aemoji <emoji>: Emoji to use for custom status when connected to network A (default "house")
  -atext string
        -atext <status text>: Description to use for custom status when connected to network A (default "Working from home")
  -atime string
        -atime <hh:mm>: Time of today when to clear status when connected to network A (default "18:00")
  -bcidr string
        -bcidr <Class Inter-Doman Routing>: CIDR addrs of network B, e.g. 192.168.1.0/24 for all ip addresses in 192.168.1.*
  -bemoji string
        -bemoji <emoji>: Emoji to use for custom status when connected to network B (default "office")
  -btext string
        -btext <status text>: Description to use for custom status when connected to network B (default "At the office")
  -btime string
        -btime <hh:mm>: Time of today when to clear status when connected to network B (default "18:00")
  -token string
        -token <Mattermost User Authorization Token>
  -url string
        -url <URL of your mattermost server>
  -username string
        -username <Mattermost username>
  -password string
        -password <Mattermost password>
  -showtoken boolean
        -showtoken true|false: Wether to output the Mattermost access token to stdout (default false)
```

## Why there are a and b networks and not just one
The script is overriding the status of each networks once it connects to the other one. This may happen if you start your work at home and then travel to the office later on. To not override other custom, manually set, custom statuses, this only happens if the text message is known to the script.

## Why only two networks
This is *my* typical scenario. In case you need more networks (working at the coffee shop) you can run the script twice, but have to live with the drawbacks of not overriding the former status (see above), or simply extend this script.

# Example

Minimal usage:

```bash
workplacer -token ASCRCYX793CYHRTWDS -url https://your.mattermost-server.net -username thatsme -acidr "192.168.3.0/24" -bcidr "10.2.0.0/16"
```
# Logging in

You can provide your password via the `-password`-option. To avoid putting your password into a script use the option `-showtoken` to have your token printed to the terminal once and from there on use this instead of the password, see below.

If neither token nor password is provided on the commandline you are queried for the password. 

## Mattermost access token

To avoid having to put your password onto the commandline this script relies on an access token. 

In case you don't want to put this token as a commandline option you can also set it to the environment variable `MATTERMOST_TOKEN`.

You can use the password option of the script to retrieve the token or query the mattermost server directly: https://api.mattermost.com/#tag/authentication

```
curl -i -d '{"login_id":"someone@nowhere.com","password":"thisisabadpassword"}' http://localhost:8065/api/v4/users/login
```

# Installation

You need to have a compiler for the go programming language locally installed.

```
go install github.com/stmichaelis/workplacer@latest
```

May take a while to download as it is using the official Mattermost API binding.

# Automatic run on Windows based on connection events
On Windows you can use the task scheduler to trigger a run of the script based on network connection events. Select in the trigger section to run *on an event*, log should be set to *Microsoft-Windows-NetworkProfile/Operational*, source to *NetworkProfile* and event id to *10000*.

# False positives when running the script

False positive, i.e. setting the wrong status, may occur when there is an overlap between home and work networks, you are using a VPN, or using the device at a place (e.g. coffee shops) which is neither work nor home and using the same private IP network from one of your other locations. You can try:

* For VPNs (which in most cases are for connecting to the work network) you should set your home network to the `acidr` network. The a-networks takes precedence over the b-network and even when in a VPN your home network should still be there.
* Think about using IPv6 addresses. Many providers keep the first parts of your assigned network adress constant, based on physical location.
* For WiFi-connections: Let the script only run when connected to specific networks. In the Windows task scheduler you can select the specific network on the *conditions* tab.
* Change your local home network address space. It's your network after all.