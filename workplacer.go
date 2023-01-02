package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"syscall"
	"time"

	"golang.org/x/term"

	mattermost "github.com/mattermost/mattermost-server/v6/model"
)

var (
	client         *mattermost.Client4
	user           *mattermost.User
	mattermostURL  string
	authtoken      string
	username       string
	password       string
	showtoken      bool
	acidr, bcidr   string
	aemoji, bemoji string
	atext, btext   string
	atime, btime   string
	mlog           bool
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
	flag.BoolVar(&mlog, "mlog", false, "Sends log messages to the given Mattermost user in addition to stdout if set")

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
		matterlog(err.Error(), "", true)
	}
	addr, err := net.InterfaceAddrs()
	if err != nil {
		matterlog(err.Error(), "", true)
	}
	for _, a := range addr {
		i, _, err := net.ParseCIDR(a.String())
		if err != nil {
			matterlog(err.Error(), "", true)
		}
		if ipNet.Contains(i) {
			return true
		}
	}
	return false
}

// mattermostLogin logs you into the server using presented credentials
func mattermostLogin() error {
	client = mattermost.NewAPIv4Client(mattermostURL)
	client.AuthToken = authtoken
	client.AuthType = "BEARER"

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
				return err
			}

			password = string(bytePassword)
		}
		user, _, err = client.Login(username, password)
	}

	if err != nil {
		return err
	}

	// Print authentication token for reuse in subsequent queries instead of password
	if showtoken {
		fmt.Println(client.AuthToken)
	}

	return nil
}

// activateStatus sets the custom status message using the give "emoji" identifier and message "text".
// Set expiration to "times"
func activateStatus(emoji, text, times string) {
	if client == nil || user == nil {
		err := mattermostLogin()
		if err != nil {
			log.Fatalln(err)
		}
	}

	status := user.GetCustomStatus()
	if status == nil {
		status = &mattermost.CustomStatus{}
	}
	// Status different from status text set here => Keep current status
	if status.AreDurationAndExpirationTimeValid() && status.Text != atext && status.Text != btext && status.Text != "" {
		matterlog("Found status text: %v. Keeping current status.\n", status.Text, false)
		return
	}

	// Calculate expiration date for status
	now := time.Now()
	loc := now.Location()

	tofexpiry, err := time.ParseInLocation("15:04", times, loc)
	if err != nil {
		matterlog(err.Error(), "", true)
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
		matterlog(err.Error(), "", true)
	}
	matterlog("Status successfully set to: %v\n", status.Text, false)
}

// matterlog sends the log message to the account set by "username" in parallel to stdout
func matterlog(message, info string, fatal bool) {
	var msg string
	var err error
	if info != "" {
		msg = fmt.Sprintf(message, info)
	} else {
		msg = message
	}
	// Logging direct message to username on Mattermost server
	if mlog {
		if client == nil || user == nil {
			err = mattermostLogin()
		}
		if err != nil {
			log.Println(err)
		} else {
			channel, _, err := client.CreateDirectChannel(user.Id, user.Id)
			if err != nil {
				log.Println(err)
			} else {
				post := &mattermost.Post{
					UserId:    user.Id,
					ChannelId: channel.Id,
					Message:   msg,
				}
				_, _, err = client.CreatePost(post)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
	// Console log
	if !fatal {
		log.Print(msg)
	} else {
		log.Fatalln(msg)
	}
}

func main() {
	if isInNetwork(acidr) {
		activateStatus(aemoji, atext, atime)
	} else if isInNetwork(bcidr) {
		activateStatus(bemoji, btext, btime)
	} else {
		matterlog("Not in range of given networks. Nothing done.\n", "", false)
	}
}
