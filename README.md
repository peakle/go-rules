# go-rules

Go rules linter

[![Tests](https://github.com/peakle/go-rules/workflows/Tests/badge.svg)](https://github.com/peakle/go-rules/blob/master/.github/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/peakle/go-rules)](https://goreportcard.com/report/github.com/peakle/go-rules)
[![Go Reference](https://pkg.go.dev/badge/github.com/peakle/go-rules.svg)](https://pkg.go.dev/github.com/peakle/go-rules)

### How to use:

Full installation example: https://github.com/peakle/dc-rules-example

1. Install rules:
      ```shell
      go get -v github.com/peakle/go-rules
      ```
2. Create rules.go file in your project like
   in [example](https://github.com/peakle/go-rules/tree/master/example/rules.go)
3. Add linter to your pipeline:
    1. Like another one check in golangci-lint (will work for golangci-lint >v1.46.0):

        ``` yaml
        linters:
          enable:
            - gocritic
        linters-settings:
          gocritic:
            enabled-checks:
              - ruleguard
            settings:
              ruleguard:
                rules: "YourDir/rules.go"
        ```
    2. Like explicit check WITHOUT golangci-lint:
        1. install binary by `curl` or `go install`:
           ```shell
           curl -sSfL https://raw.githubusercontent.com/peakle/go-rules/master/releaser/install.sh | sh -s -- -d -b $(go env GOPATH)/bin latest
           ```
           OR
           ``` shell
           go install -v github.com/peakle/go-rules/cmd/goRules@latest
           ```
        2. Run lint:
           ``` shell
           goRules ./...
           ```
    3. Like file watcher in Goland IDE (will work for golangci-lint >v1.46.0):
        1. add golangci-lint as `File Watcher` in IDE (Preferences -> Tools -> File Watchers -> Add)
        2. set `Arguments` field where `.golangci.yml` file will be like example above:
           ```
           run $FileDir$ --config=$ProjectFileDir$/.golangci.yml
           ```

### How to update to new rules version:

1. update rules version in your go.mod file
2. download new rules version:
   ```shell
   go get github.com/peakle/go-rules@newVersion
   ```
3. if you using golangci-lint update cache:
   ```shell
   golangci-lint cache clean
   ```

### How to add new checks:

1. Ruleguard tour for newbees: https://go-ruleguard.github.io/by-example
2. Fork repo && open PR :D
