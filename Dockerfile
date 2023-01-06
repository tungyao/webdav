FROM golang:1.19
WORKDIR /home
COPY go.mod ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /home/webdav ./...
ENV UNAME = "zxc"
ENV UPASS = "zxc"
ENV MAXERR = "0"
CMD "/home/webdav" "-uname=${UNAME}" "-upass=${UPASS}" "-maxerr=${MAXERR}"
# docker run -d --name=webdav --restart=always -v /mnt:/mnt -v /home/dong:/home/dong -p 10000:80 -e UNAME=zxc -e UPASS=zxc -e MAXERR=0 tungyao/webdav