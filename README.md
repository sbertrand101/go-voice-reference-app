# go-voice-reference-app

[![Build](https://travis-ci.org/BandwidthExamples/go-voice-reference-app.png)](https://travis-ci.org/BandwidthExamples/go-voice-reference-app)

 This application demonstrates how to implement voice calling for mobile devices, browsers (WebRTC), and any SIP client using the [Catapult API](http://ap.bandwidth.com/?utm_medium=social&utm_source=github&utm_campaign=dtolb&utm_content=_).
    This reference application makes creating, registering, and implementing voice calling for endpoints (mobile, web, or any SIP client) easy.
    This application implements the steps documented [here](http://ap.bandwidth.com/docs/how-to-guides/use-endpoints-make-receive-calls-sip-clients/).

You can open up the web page at the root of the deployed project for more instructions and for example of voice calling in your web browser using WebRTC.

Uses the:
* [Catapult Golang SDK](https://github.com/bandwidthcom/go-bandwidth/?utm_medium=social&utm_source=github&utm_campaign=dtolb&utm_content=_)

## Prerequisites
- Configured Machine with Ngrok/Port Forwarding -OR- Heroku Account
  - [Ngrok](https://ngrok.com/)
  - [Heroku](https://www.heroku.com/)
- [Catapult Account](http://ap.bandwidth.com/?utm_medium=social&utm_source=github&utm_campaign=dtolb&utm_content=_)
- [PostgreSQL](http://www.postgresql.org/download/)
- [Go 1.6+](https://golang.org/dl/)

## Deploy To PaaS

#### Env Variables Required To Run
* ```CATAPULT_USER_ID```
* ```CATAPULT_API_TOKEN```
* ```CATAPULT_API_SECRET```

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

## Install
Before running export next environment variables :

```CATAPULT_USER_ID```, ```CATAPULT_API_TOKEN```, ```CATAPULT_API_TOKEN``` - auth data for Catapult API (to search and reserve a phone number, etc)

Set environment variable `DATABASE_URL` with connection string to existing PostgresSQL database.

Install `godep` via `go get github.com/tools/godep` if need.

After that run `godep go build`  to build executable file.

You can run this demo  like `./go-voice-reference-app` (use environment variable `PORT` to change port to listen to) on local machine if you have ability to handle external requests or use any external hosting.

## Deploy on Heroku

Create account on [Heroku](https://www.heroku.com/) and install [Heroku Toolbel](https://devcenter.heroku.com/articles/getting-started-with-go#set-up) if need.


Run `heroku create` to create new app on Heroku and link it with current project.

Configure the app by commands

```
 heroku config:set CATAPULT_USER_ID=your-user-id
 heroku config:set CATAPULT_API_TOKEN=your-token
 heroku config:set CATAPULT_API_SECRET=your-secret
```

Add PostgreSQL support by

```
heroku addons:create heroku-postgresql:hobby-dev
```

Run `git push heroku master` to deploy this project.

Run `heroku open` to see home page of the app in the browser

