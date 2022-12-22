package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
	"syscall"

    "golang.org/x/term"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

var (
	mattermostURL  string
	authtoken      string
	username       string
	password       string
	showtoken      bool
	acidr, bcidr   string
	aemoji, bemoji string
	atext, btext   string
	atime, btime   string
)

func init() {
	flag.StringVar(&mattermostURL, "url", "", "-url <URL of your mattermost server>")
	flag.StringVar(&authtoken, "token", "", "-token <Mattermost User Authorization Token>")
	flag.StringVar(&username, "username", "", "-username <Mattermost username without leading @>")
	flag.StringVar(&acidr, "acidr", "", "-acidr <Class Inter-Doman Routing>: CIDR address of network A, e.g. 192.168.1.0/24 for all ip addresses in 192.168.1.*")
	flag.StringVar(&bcidr, "bcidr", "", "-bcidr <Class Inter-Doman Routing>: CIDR addrs of network B, e.g. 192.168.1.0/24 for all ip addresses in 192.168.1.*")
	flag.StringVar(&aemoji, "aemoji", "house", "-aemoji <emoji>: Emoji to use for custom status when connected to network A")
	flag.StringVar(&bemoji, "bemoji", "office", "-bemoji <emoji>: Emoji to use for custom status when connected to network B")
	flag.StringVar(&atext, "atext", "Working from home", "-atext <status text>: Description to use for custom status when connected to network A")
	flag.StringVar(&btext, "btext", "At the office", "-btext <status text>: Description to use for custom status when connected to network B")
	flag.StringVar(&atime, "atime", "18:00", "-atime <hh:mm>: Time of today when to clear status when connected to network A")
	flag.StringVar(&btime, "btime", "18:00", "-btime <hh:mm>: Time of today when to clear status when connected to network B")
	flag.StringVar(&password, "password", "", "-password <Password of your Mattermost account>. Reads from stdin if set to \"-\" or empty and no authenticaton token set.")
	flag.BoolVar(&showtoken, "showtoken", false, "Wether to output the Mattermost access token to stdout")

	flag.Parse()

	if authtoken == "" {
		authtoken = os.Getenv("MATTERMOST_TOKEN")
	}
}

// isInNetwork returns if the running computer is connected to the network identified by "cidr"
func isInNetwork(cidr string) bool {
	// Empty netmasks do not count as errors
	if cidr == "" {
		return false
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Fatalln(err)
	}
	addr, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalln(err)
	}
	for _, a := range addr {
		i, _, err := net.ParseCIDR(a.String())
		if err != nil {
			log.Fatalln(err)
		}
		if ipNet.Contains(i) {
			return true
		}
	}
	return false
}

// activateStatus sets the custom status message using the give "emoji" identifier and message "text".
// Set expiration to "times"
func activateStatus(emoji, text, times string) {
	client := mattermost.NewAPIv4Client(mattermostURL)
	client.AuthToken = authtoken
	client.AuthType = "BEARER"

	var user *mattermost.User
	var err error

	// Use authorization token only if not overriden by password
	if authtoken != "" && password == "" {
		user, _, err = client.GetUserByUsername(username, "")
	} else {
		if password == "" || password == "-" {
			// Reading password from commandline
			fmt.Print("Password: ")
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
    		if err != nil {
        		log.Fatalln(err)
    		}

    		password = string(bytePassword)
		}
		user, _, err = client.Login(username, password)
	}

	if err != nil {
		log.Fatalln(err)
	}

	// Print authentication token for reuse in subsequent queries instead of password
	if showtoken {
		fmt.Println(client.AuthToken)
	}

	status := user.GetCustomStatus()
	if status == nil {
		status = &mattermost.CustomStatus{}
	}
	// Status different from status text set here => Keep current status
	if status.AreDurationAndExpirationTimeValid() && status.Text != atext && status.Text != btext && status.Text != "" {
		log.Printf("Found status text: %v. Keeping current status.\n", status.Text)
		return
	}

	// Calculate expiration date for status
	now := time.Now()
	loc := now.Location()

	tofexpiry, err := time.ParseInLocation("15:04", times, loc)
	if err != nil {
		log.Fatalln(err)
	}
	tofexpiry = time.Date(now.Year(), now.Month(), now.Day(), tofexpiry.Hour(), tofexpiry.Minute(), 0, 0, loc)

	// Set new status
	status.Duration = "date_and_time"
	status.Emoji = emoji
	status.Text = text
	status.ExpiresAt = tofexpiry
	status.PreSave()
	_, _, err = client.UpdateUserCustomStatus(user.Id, status)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Status successfully set to: %v\n", status.Text)
}

func main() {
	if isInNetwork(acidr) {
		activateStatus(aemoji, atext, atime)
	} else if isInNetwork(bcidr) {
		activateStatus(bemoji, btext, btime)
	} else {
		log.Println("Not in range of given networks. Nothing done.")
	}
}
