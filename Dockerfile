FROM debian
WORKDIR /home
COPY webdav webdav
RUN chmod +x /home/webdav
ENV UNAME = "zxc"
ENV UPASS = "zxc"
CMD "/home/webdav" "-uname=${UNAME}" "-upass=${UPASS}"