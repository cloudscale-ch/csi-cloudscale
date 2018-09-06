# cloudscale.ch Go API SDK

If you want to manage your cloudscale.ch server resources with Go, you are at
the right place.

There's a possibility to specify the `CLOUDSCALE_URL` environment variable to
change the default url of https://api.cloudscale.ch.

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
CLOUDSCALE_TOKEN="HELPIMTRAPPEDINATOKENGENERATOR" make integration

````

If you want to give params to `go test`, you can use something like this:
```
TESTARGS='-run FloatingIP' make integration
```
