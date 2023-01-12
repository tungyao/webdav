FROM ubuntu
WORKDIR /home
COPY ./webdav /home/webdav
COPY ./main.db /home/webdav/db/main.db
RUN chmod +x /home/webdav
ENV UNAME = "zxc"
ENV UPASS = "zxc"
ENV MAXERR = "0"
CMD "/home/webdav" "-uname=${UNAME}" "-upass=${UPASS}" "-maxerr=${MAXERR}"
# docker run -d --name=webdav --restart=always -v /mnt:/mnt -v /home/dong:/home -p 10000:80 -e UNAME=zxc -e UPASS=zxc -e MAXERR=0 tungyao/webdav