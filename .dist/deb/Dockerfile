FROM ubuntu:jammy
ARG DEBIAN_FRONTEND=noninteractive

# Set the working directory to /app
WORKDIR /opt/bw/.dist/deb

RUN apt-get update
RUN apt-get -y install dput devscripts dh-make dput golang-1.18

CMD /bin/bash build.sh focal jammy
