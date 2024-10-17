e2e-setup: clean
    go run ./e2e-tests/run/main.go --name no_server

clean:
    rm -rf ./e2e-tests/data/*

e2e-debug:
    DEBUG_LOG=/tmp/mm-testing GAME_SERVER="{{justfile_directory()}}/cmd/api-server/main.go" go test -v ./e2e-tests/... &
    tail -F /tmp/mm-testing &
    wait

e2e:
    GAME_SERVER="{{justfile_directory()}}/cmd/api-server/main.go" go test ./e2e-tests/...

kill-tests:
    ps aux | grep "go test" | grep -v "grep" | awk '{print $2}' | xargs -I {} kill -9 {}

sim-search-id id:
    cat err | grep ":{{id}}"  | go run ./cmd/log-parser/main.go
