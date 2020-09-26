FROM ubuntu:20.04
ARG DEBIAN_FRONTEND=noninteractive

# Set the working directory to /app
WORKDIR /opt/bw/.dist/deb

RUN apt-get update
RUN apt-get -y install dput devscripts dh-make golang-1.13 dput
CMD /bin/bash build.sh focal
