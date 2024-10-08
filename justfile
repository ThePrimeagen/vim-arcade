e2e: clean
    go run ./e2e-tests/run/main.go --name no_server

clean:
    rm -rf ./e2e-tests/data/*

