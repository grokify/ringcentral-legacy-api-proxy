RingCentral Legacy API Proxy
============================

[![Build Status][build-status-svg]][build-status-link]
[![Go Report Card][goreport-svg]][goreport-link]
[![Docs][docs-godoc-svg]][docs-godoc-link]
[![License][license-svg]][license-link]

 [build-status-svg]: https://api.travis-ci.org/grokify/ringcentral-legacy-api-proxy.svg?branch=master
 [build-status-link]: https://travis-ci.org/grokify/ringcentral-legacy-api-proxy
 [goreport-svg]: https://goreportcard.com/badge/github.com/grokify/ringcentral-legacy-api-proxy
 [goreport-link]: https://goreportcard.com/report/github.com/grokify/ringcentral-legacy-api-proxy
 [docs-godoc-svg]: https://img.shields.io/badge/docs-godoc-blue.svg
 [docs-godoc-link]: https://godoc.org/github.com/grokify/ringcentral-legacy-api-proxy
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-link]: https://github.com/grokify/ringcentral-legacy-api-proxy/blob/master/LICENSE

This is a proxy service that allows apps using RingCentral's older APIs to use the new [RingCentral REST APIs](https://developer.ringcentral.com) seamlessly.

* https://service.ringcentral.com/ringoutapi/ ([docs](https://grokify.github.io/ringcentral-legacy-api-proxy/ringoutapi.html))
* https://service.ringcentral.com/faxoutapi/ ([docs](https://grokify.github.io/ringcentral-legacy-api-proxy/faxoutapi.html))

The following calls with checks are currently supported:

* [x] [RingOut `call` command](https://grokify.github.io/ringcentral-legacy-api-proxy/ringoutapi.html#call) - returns REST API JSON response
* [ ] [RingOut `list` command](https://grokify.github.io/ringcentral-legacy-api-proxy/ringoutapi.html#list)
* [ ] [RingOut `status` command](https://grokify.github.io/ringcentral-legacy-api-proxy/ringoutapi.html#status)
* [ ] [RingOut `cancel` command](https://grokify.github.io/ringcentral-legacy-api-proxy/ringoutapi.html#cancel)
* [ ] [FaxOut](https://grokify.github.io/ringcentral-legacy-api-proxy/faxoutapi.html)

## Prerequisites

The following are required to use this app.

1. Account at https://developer.ringcentral.com
2. An application that supports the "Password grant" OAuth 2.0 flow and the `RingOut` permission

## Configuration

This application needs the following configuration variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `RINGCENTRAL_CLIENT_ID` | yes | Your application's Client ID |
| `RINGCENTRAL_CLIENT_SECRET` | yes | Your application's Client Secret |
| `RINGCENTRAL_SERVER_URL` | yes | Your RingCentral server url, e.g. Sandbox: https://platform.devtest.ringcentral.com , Production: https://platform.ringcentral.com |

## Installation

### Deploying to Heroku

```sh
$ heroku create
$ git push heroku master
$ heroku open
```

or

[![Deploy](https://www.herokucdn.com/deploy/button.png)](https://heroku.com/deploy)

After you click the button above, you will need to enter the environment variables above into the Heroku web console.

### Running Locally

```
$ go get github.com/grokify/ringcentral-legacy-api-proxy
$ cd ringcentral-legacy-api-proxy
$ go get ./...
$ go build main.go
$ RINGCENTRAL_SERVER_URL=https://platform.devtest.ringcentral.com \
  RINGCENTRAL_CLIENT_ID=<myClientId> \
  RINGCENTRAL_CLIENT_SECRET=<myClientSecret> \
  main
```

## Notes

Rebuild `vendor` directory with:

```
$ godep save ./...
```

More information on deploying Go on Heroku here:

* https://devcenter.heroku.com/articles/go-support