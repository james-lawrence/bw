FROM ubuntu:20.04
ARG DEBIAN_FRONTEND=noninteractive

# Set the working directory to /app
WORKDIR /opt/bw/.dist/deb

RUN apt-get update
RUN apt-get -y install dput devscripts dh-make golang-1.14 dput
RUN cp /usr/lib/go-1.14/bin/go /usr/bin/go
RUN go version
CMD /bin/bash build.sh focal
