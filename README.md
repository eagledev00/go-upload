# go-upload

Simple go app for file uploads.

This has a rudimentary web UI if needed and turned on with `ENABLE_WEBFORM` environment variable
and it can serve the files from uploaded directory.

However, I mostly use nginx to proxy the public url into storage path

## Docker images

https://hub.docker.com/r/irwiss/go-upload/tags

## Environment variables

| Variable              | Example Value               | Description                                                                                                       |
|-----------------------|-----------------------------|-------------------------------------------------------------------------------------------------------------------|
| LISTEN_ADDRESS        | 0.0.0.0:9000                | Endpoint to bind the http server, example value will bind to any ipv4 address at port 9000                        |
| MAX_UPLOAD_SIZE_IN_MB | 100                         | Value of `100` means files larger than 100MB will be rejected                                                     |
| STORAGE_PATH          | ./                          | Value of `./` will put the files wherever working directory is set to                                             |
| FILENAME_LENGTH       | 8                           | Value of `8` will pick 8 random alphanumeric characters as filename + the file's extension as the result filename |
| ENABLE_WEBFORM        | true                        | Value of `true` will let you see a simple web UI for uploadnig                                                    |
| PUBLIC_ROOT           | http://example.com/uploads/ | The url where your STORAGE_PATH is visible from public internet, with trailing slash                              |
