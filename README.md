# simple-mock-server
Local server to mock API calls to an external service.

# Installation
## Env variables
Set SIMPLE_MOCKS_LOCATION environment variable with the directory of your json files.

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
            "key": "value", 
            "anotherkey": 123
        },
        "discard_query_params": false
    }


## Run
`go run ./...` in the proyect root.