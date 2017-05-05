# lgtm; slackbot for #pr-party [![Build Status](https://travis-ci.org/StudentRND/lgtm.svg?branch=master)](https://travis-ci.org/StudentRND/lgtm)

lgtm lets users add github integrations to the slack channel #pr-party for
posting to the channel when a new pull request opens.

## TODO

- [ ] OAuth2 flow with github to be able to make web hooks on a users repos
	- [ ] [github.com/golang/oauth2](https://github.com/golang/oauth2)
	- [ ] Any OAuth token
	- [ ] Associate slack user with OAuth2 token
- [ ] Github Webhooks
	- [ ] [github.com/google/go-github](https://github.com/google/go-github)
	- [ ] Store slack webhook
	- [ ] Add webhook to repo
		- [ ] application/json
		- [ ] Pull request
- [ ] Bot
	- [ ] Connect to slack
		- [X] Do random stuff
		- [ ] Move code out of `main.go` to `lgtm/bot`
	- [ ] Flow that does oauth
		- [ ] `@lgtm Authenicate me`
	- [ ] Flow that adds webhook
		- [ ] `@lgtm Watch repo adamryman/kit`
	- [ ] Flow that removes webhook?
		- [ ] `@lgtm stop looking at adamryman/kit`
	- [ ] Parse webhook response, post in #pr-party
		- [ ] Look for `{ "action": "opened"}`
			- [ ] Post in #pr-party
		- [ ] Look for `{ "action": "closed"}`
			- [ ] React to old post in #pr-party

## (UN)LICENSE

[This is free and unencumbered software released into the public domain.](./UNLICENSE)
