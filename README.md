# simple-mock-server
Local server to mock API calls to an external service.

# Installation
## Env variables
Set SIMPLE_MOCKS_LOCATION environment variable with the directory of your json files or use the -d flag.

REMEMBER TO USE THE .json EXTENSION in each mock!

## JSON mock format
    {
        "paths": ["/some/hardcoded/path", "/another/hardcoded/path"],
        "verb": "GET",
        "body": {
            "user": 123,
            "name": "bk9"
        },
        "headers": {
            "Content-Type": "application/json",
            "key": "value", 
            "anotherkey": "123"
        },
        "status": 267
    }

- Paths: paths to match.
- Verb: REST verb to match.
- Body: Response body of the API.
- Headers: Response headers of the API.
- Status: Status code to return.

## Run
`go run ./cmd/api/main.go -p=<port> -d=<path-to-mocks>` in the proyect root.
