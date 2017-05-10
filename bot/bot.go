package bot

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nlopes/slack"
	"github.com/pkg/errors"

	_ "github.com/joho/godotenv/autoload"
	. "github.com/y0ssar1an/q"
)

const prPartyID = "C3YJF4GP5"
const playgroundId = "C03LPQF0Y"

var slackChannel = prPartyID

func init() {
	Q(os.Getenv("DEBUG"))
	if len(os.Getenv("DEBUG")) > 0 {
		slackChannel = playgroundId
	}
}

const lgtmID = "<@U456ZSLSJ>"

type LGTM struct {
	api                  *slack.Client
	emoji                string
	logger               *log.Logger
	RequestRepoWatchHook func(ctx context.Context, owner, repo string)
	IncomingEvents       chan interface{}
}

type WatchRepoEvent struct {
	User  string
	Owner string
	Repo  string
}

// LGTMOption is a function that modifies the client config
type LGTMOption func(*LGTM) error

func SetLogger(logger *log.Logger) LGTMOption {
	return func(o *LGTM) error {
		o.logger = logger
		return nil
	}
}

func SetRepoWatchHook(hook func(ctx context.Context, owner, repo string)) LGTMOption {
	return func(o *LGTM) error {
		o.RequestRepoWatchHook = hook
		return nil
	}
}

func Start(ctx context.Context, token string, options ...LGTMOption) (*LGTM, error) {
	lgtm := LGTM{
		IncomingEvents: make(chan interface{}),
	}

	for _, f := range options {
		err := f(&lgtm)
		if err != nil {
			return nil, errors.Wrap(err, "cannot apply option")
		}
	}

	slack.SetLogger(lgtm.logger)
	lgtm.api = slack.New(token)
	go lgtm.start(ctx, token)

	return &lgtm, nil
}

func (lgtm LGTM) PostMessage(msg string) (timestamp string, err error) {
	pmp := slack.NewPostMessageParameters()
	pmp.AsUser = true
	pmp.LinkNames = 1
	pmp.EscapeText = false
	_, timestamp, err = lgtm.api.PostMessage(slackChannel, msg, pmp)
	return
}

//func (lgtm LGTM) PostPullRequest(url string) (timestamp string, err error) {
//pmp := slack.NewPostMessageParameters()
//pmp.AsUser = true
//_, timestamp, err = lgtm.api.PostMessage(prPartyID, url, pmp)
//return
//}

func (lgtm LGTM) ReactPullRequest(timestamp string) error {
	pmp := slack.NewPostMessageParameters()
	pmp.AsUser = true
	itemref := slack.NewRefToMessage(slackChannel, timestamp)
	return lgtm.api.AddReaction(lgtm.emoji, itemref)
}

func (lgtm *LGTM) start(ctx context.Context, token string) {

	lgtm.api.SetDebug(true)

	rtm := lgtm.api.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		fmt.Print("Event Received: ")
		switch ev := msg.Data.(type) {
		case *slack.HelloEvent:
			// Ignore hello
		case *slack.ChannelJoinedEvent:
			fmt.Println(ev.Channel.ID)

		case *slack.ConnectedEvent:
			fmt.Println("Infos:", ev.Info)
			fmt.Println("Connection counter:", ev.ConnectionCount)

		case *slack.MessageEvent:
			text := ev.Text
			if ev.Channel != slackChannel {
				continue
			}
			if !strings.Contains(text, lgtmID) {
				continue
			}
			searchText := strings.ToLower(ev.Text)
			Q(searchText)
			watchRequest := strings.Split(searchText, " watch repo ")
			switch {
			case len(watchRequest) > 1:
				Q(ev.User)
				Q(watchRequest)
				repoPart := watchRequest[len(watchRequest)-1]
				go lgtm.handleWatchRequest(ev, repoPart)
			}

			fmt.Printf("Message: %v\n", ev)

		case *slack.PresenceChangeEvent:
			fmt.Printf("Presence Change: %v\n", ev)

		case *slack.LatencyReport:
			fmt.Printf("Current latency: %v\n", ev.Value)

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")

		default:

			// Ignore other events..
			// fmt.Printf("Unexpected: %v\n", msg.Data)
		}
	}
}

func (lgtm *LGTM) handleWatchRequest(msg *slack.MessageEvent, repoPart string) {
	Q(repoPart)
	scanner := bufio.NewScanner(strings.NewReader(repoPart))
	scanner.Split(bufio.ScanWords)
	if !scanner.Scan() {
		return
	}

	repoText := scanner.Text()
	ownerRepo := strings.Split(repoText, "/")
	Q(repoText, ownerRepo)
	if len(ownerRepo) < 2 {
		return
	}
	owner, repo := ownerRepo[0], ownerRepo[1]
	Q(owner, repo)
	lgtm.IncomingEvents <- WatchRepoEvent{User: msg.User, Owner: owner, Repo: repo}
}
