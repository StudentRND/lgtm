package lgtm

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/StudentRND/lgtm/bot"
	"github.com/StudentRND/lgtm/github"
	sql "github.com/StudentRND/lgtm/sqlite"

	_ "github.com/joho/godotenv/autoload"
	. "github.com/y0ssar1an/q"
)

func init() {

}

func Start() error {
	database, err := sql.Open(os.Getenv("SQLITE3"))
	_ = database

	slackWebhook := os.Getenv("SLACK_WEBHOOK")
	slackToken := os.Getenv("SLACK_API_TOKEN")
	githubToken := os.Getenv("GITHUB_TOKEN")
	Q(githubToken)

	Q(slackToken)
	logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	ctx := context.Background()

	lgtm, err := bot.Start(ctx, slackToken, bot.SetLogger(logger))
	if err != nil {
		return err
	}
	lgtm.PostMessage("Rebooting....Done " + time.Now().String())

	// handle chat events
	go func() {
		for event := range lgtm.IncomingEvents {
			switch ev := event.(type) {
			case bot.WatchRepoEvent:
				lgtm.PostMessage(fmt.Sprintf("someday I will watch `%s` with owner `%s`, <@%s>", ev.Repo, ev.Owner, ev.User))
				token, err := database.ReadUserAuth(ev.User)
				if err != nil {
					lgtm.PostMessage(fmt.Sprintf("You need to authorize me <@%s>. http://home.adamryman.com:5040/authorize?slack_id=%s", ev.User, ev.User))
					//token = githubToken
					continue
				}
				hook, err := github.WatchRepo(ctx, token, ev.Owner, ev.Repo)
				if err != nil {
					fmt.Println(err)
					Q(err)
					continue
				}
				lgtm.PostMessage(fmt.Sprintf("And that day is today! Look at that! %s", *hook.URL))
				Q(hook)
			}
		}
	}()

	// start github handlers
	go func() {
		github.SlackWebhook = slackWebhook
		http.HandleFunc("/authorize", github.AuthenicateHandler)
		http.HandleFunc("/authorize/callback", github.AuthenticateCallbackHandler)
		http.HandleFunc("/webhook", github.WebhookHandler)
		http.ListenAndServe(":5040", nil)
	}()

	// handle github events
	go func() {
		for event := range github.IncomingEvents {
			switch ev := event.(type) {
			case github.AuthenticateEvent:
				err := database.CreateUser(ev.SlackId, ev.Token)
				if err != nil {
					Q(err)
					continue
				}
				lgtm.PostMessage(fmt.Sprintf("<@%s> has Authenticated", ev.SlackId))
			case github.PullRequestEvent:
				switch ev.Action {
				case "open":
					ts, err := lgtm.PostMessage(ev.URL)
					if err != nil {
						Q(err)
						continue
					}
					_ = ts
					// TODO: Store pull request id and timestamp
				case "close":
					// TODO: pull out time for pull request id and react to that message
					//lgtm.ReactPullRequest()
				}
			}
		}

	}()

	// Wait for termination
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	lgtm.PostMessage("TERMINATED")
	time.Sleep(time.Second)

	return nil
}
