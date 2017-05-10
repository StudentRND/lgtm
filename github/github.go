package github

import (
	"context"
	"io/ioutil"
	"math"
	"net/http"
	"os"

	"github.com/google/go-github/github"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	github_goth "github.com/markbates/goth/providers/github"
	"golang.org/x/oauth2"

	. "github.com/y0ssar1an/q"
)

import _ "github.com/joho/godotenv/autoload"

var SlackWebhook string

func init() {
	SlackWebhook = os.Getenv("SLACK_WEBHOOK")
	Q(SlackWebhook)
}

var IncomingEvents chan interface{}

func init() {
	IncomingEvents = make(chan interface{})
}

type PullRequestEvent struct {
	Id     int
	Action string
	URL    string
}

type AuthenticateEvent struct {
	SlackId string
	Token   string
}

func init() {
	store := sessions.NewFilesystemStore(os.TempDir(), []byte("goth-example"))

	// set the maxLength of the cookies stored on the disk to a larger number to prevent issues with:
	// securecookie: the value is too long
	// when using OpenID Connect , since this can contain a large amount of extra information in the id_token

	// Note, when using the FilesystemStore only the session.ID is written to a browser cookie, so this is explicit for the storage on disk
	store.MaxLength(math.MaxInt64)

	gothic.Store = store
}

func init() {
	clientId := os.Getenv("GITHUB_OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_OAUTH_CLIENT_SECRET")
	redirectURL := os.Getenv("GITHUB_OAUTH_REDIRECT")

	gothic.GetProviderName = func(req *http.Request) (string, error) { return "github", nil }
	provider := github_goth.New(clientId, clientSecret, redirectURL, "admin:repo_hook")
	goth.UseProviders(provider)
}

func AuthenticateCallbackHandler(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		return
	}
	Q(user)

	// TODO: timeout
	ae := AuthenticateEvent{
		SlackId: r.URL.Query().Get("state"),
		Token:   user.AccessToken,
	}

	IncomingEvents <- ae
}

func AuthenicateHandler(w http.ResponseWriter, r *http.Request) {
	slackId := r.URL.Query().Get("slack_id")
	Q(slackId)
	oldQuery := r.URL.Query()
	oldQuery.Set("state", slackId)
	r.URL.RawQuery = oldQuery.Encode()

	Q(r.URL.Query().Get("state"))
	gothic.SetState(r)

	gothic.BeginAuthHandler(w, r)
}

// Wehook endpointu
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	//s.webhookSecretKey
	// TODO validate payload
	// https://developer.github.com/webhooks/securing/

	//payload, err := github.ValidatePayload(r, nil)
	//if err != nil {
	//Q(err)
	//}
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Q(err)
	}
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		Q(err)
		return
	}
	switch event := event.(type) {
	case *github.PullRequestEvent:
		switch action := event.GetAction(); action {
		case "open", "closed":
			// TODO: timeout
			pre := PullRequestEvent{
				Id:     event.PullRequest.GetID(),
				Action: event.GetAction(),
				URL:    event.PullRequest.GetURL(),
			}
			IncomingEvents <- pre
		}
	}
}

func WatchRepo(ctx context.Context, token, owner, repo string) (*github.Hook, error) {
	req := new(github.Hook)
	name := "web"
	active := true
	req = &github.Hook{
		Name:   &name,
		Active: &active,
		Events: []string{"pull_request"},
		Config: map[string]interface{}{
			"url":          SlackWebhook,
			"content_type": "json",
		},
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		},
	)

	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	hook, _, err := client.Repositories.CreateHook(ctx, owner, repo, req)
	return hook, err
}
