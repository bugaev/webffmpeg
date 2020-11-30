# webffmpeg

This is a web interface for vidstab plugin of ffmpeg.

Upload your shaky video file, get a nice one, with high frequency vibrations removed.

Video processing takes between 1 and 2 minutes on an average laptop. A real-time progres update is shown while you are waiting.
The percentage shown is relative to the original file size. Usually your result is 20%-30% smaller, so you will be done faster.

# Bare metal
To run the service on bare metal: `go build serv.go && ./serv` 

# Docker Container
```
cd DOCKER_BUILD
go mod init serv.go
go mod tidy
docker build .
```

Image ID is based on `docker image ls`, yours may be different:
```
docker run -p 127.0.0.1:8080:8080/tcp 4fe27ad1cbea /serv
```

# In browser

URL: http://localhost:8080/
