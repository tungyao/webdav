FROM ubuntu
WORKDIR /home
COPY webdav webdav
RUN chmod +x /home/webdav
ENV UNAME = "zxc"
ENV UPASS = "zxc"
ENV MAXERR = "0"
CMD "/home/webdav" "-uname=${UNAME}" "-upass=${UPASS}" "-maxerr=${MAXERR}"
# docker run -d --restart=always -v /mnt:/mnt -p 10000:80 -e UNAME=zxc -e UPASS=zxc tungyao/webdav