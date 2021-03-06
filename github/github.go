package github

import (
	"context"
	"fmt"
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

var PRWebhook string

func init() {
	PRWebhook = os.Getenv("PR_WEBHOOK")
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
	Id    string
	Token string
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

func InitAuth(clientId, clientSecret, redirectURL string) {
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
		Id:    r.URL.Query().Get("state"),
		Token: user.AccessToken,
	}

	IncomingEvents <- ae
	fmt.Fprintln(w, "Go back to slack now plz.")
}

func AuthenicateHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	Q(id)

	oldQuery := r.URL.Query()
	oldQuery.Set("state", id)
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
	Q("INCOMING WEBHOOK")

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Q(err)
	}

	Q(string(payload))

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		Q(err)

		return
	}

	switch event := event.(type) {
	case *github.PullRequestEvent:
		switch action := event.GetAction(); action {
		case "opened", "closed":

			Q("PR Opened or closed")
			// TODO: timeout

			pre := PullRequestEvent{
				Id:     event.PullRequest.GetID(),
				Action: event.GetAction(),
				URL:    event.PullRequest.GetHTMLURL(),
			}
			IncomingEvents <- pre

			Q("Event sent")
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
			"url":          PRWebhook,
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
