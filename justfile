e2e-setup: clean
    go run ./e2e-tests/run/main.go --name no_server

clean:
    rm -rf ./e2e-tests/data/*

e2e-debug:
    DEBUG_LOG=/tmp/mm-testing DUMMY_SERVER="{{justfile_directory()}}/cmd/dummy-server/main.go" go test -v ./e2e-tests/... &
    tail -F /tmp/mm-testing &
    wait

e2e:
    DUMMY_SERVER="{{justfile_directory()}}/cmd/dummy-server/main.go" go test ./e2e-tests/...
