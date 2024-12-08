FROM ubuntu:jammy
ARG DEBIAN_FRONTEND=noninteractive

# Set the working directory to /app
WORKDIR /opt/bw/.dist/deb

RUN apt-get update
RUN apt-get -y install software-properties-common build-essential
RUN add-apt-repository -n ppa:longsleep/golang-backports

RUN apt-get update
RUN apt-get -y install dput devscripts dh-make dput golang-1.22 pinentry-tty

CMD /bin/bash build.sh jammy noble oracular
