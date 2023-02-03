FROM ubuntu
WORKDIR /home
COPY ./webdav /home/webdav
COPY ./main.db /home/db/main.db
RUN chmod +x /home/webdav
ENV UNAME = "zxc"
ENV UPASS = "zxc"
ENV MAXERR = "0"
CMD "/home/webdav" "-uname=${UNAME}" "-upass=${UPASS}" "-maxerr=${MAXERR}"
# docker run -d --name=webdav --restart=always -v /mnt:/mnt -v /home/dong/webdav:/home/db -p 8049:80 -e UNAME=zxc -e UPASS=zxc -e MAXERR=0 tungyao/webdav:v1.9