FROM ubuntu:19.10

# Set the working directory to /app
WORKDIR /opt/bw/.dist/deb

RUN apt-get update
RUN apt-get -y install dput devscripts dh-make golang-1.13 dput
RUN cp /usr/lib/go-1.13/bin/go /usr/bin/go
RUN go version
CMD /bin/bash build.sh eoan
