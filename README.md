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

* https://service.ringcentral.com/ringoutapi/
* https://service.ringcentral.com/faxoutapi/

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

More information on deploying Go on Heroku here:

* https://devcenter.heroku.com/articles/go-support

## Notes

Rebuild `vendor` directory with:

```
$ godep save ./...
```