# cloudscale.ch Go API SDK
[![Go Reference](https://pkg.go.dev/badge/github.com/cloudscale-ch/cloudscale-go-sdk.svg)](https://pkg.go.dev/github.com/cloudscale-ch/cloudscale-go-sdk)
[![Tests](https://github.com/cloudscale-ch/cloudscale-go-sdk/actions/workflows/test.yaml/badge.svg)](https://github.com/cloudscale-ch/cloudscale-go-sdk/actions/workflows/test.yaml)

If you want to manage your cloudscale.ch server resources with Go, you are at
the right place.

There's a possibility to specify the `CLOUDSCALE_API_URL` environment variable to
change the default url of https://api.cloudscale.ch, but you can almost certainly
use the default.

## Download from Github

```console
GO111MODULE=on go get github.com/cloudscale-ch/cloudscale-go-sdk
```

## Testing

The test directory contains integration tests, aside from the unit tests in the
root directory. While the unit tests suite runs very quickly because they
don't make any network calls, this can take some time to run.

### test/integration

This folder contains tests for every type of operation in the cloudscale.ch API
and runs tests against it.

Since the tests are run against live data, there is a higher chance of false
positives and test failures due to network issues, data changes, etc.

Run the tests using:

````
CLOUDSCALE_API_TOKEN="HELPIMTRAPPEDINATOKENGENERATOR" make integration

````

If you want to give params to `go test`, you can use something like this:
```
TESTARGS='-run FloatingIP' make integration
```

## Releasing

To create a new release, please do the following:
 * Merge all feature branches into a release branch
 * Checkout the release branch
 * Run `make NEW_VERSION=v1.x.x bump-version`
 * Commit the changes
 * Merge the release branch into master
 * Create a [new release](https://github.com/cloudscale-ch/cloudscale-go-sdk/releases/new) on GitHub.
