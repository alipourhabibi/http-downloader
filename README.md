# http-downloader
A very simple http downloader written with syscall, which can handle downloading in chunks

## compose
you first need to put your files in /tmp/nginx
```sh
$ docker compose up -d
$ docker exec -it HTTPDL_CONTAINER sh
# /bin/httpdl --port PORT --ip IP --filename FILENAME --source true
```
