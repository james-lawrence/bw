FROM ubuntu:19.04

# Set the working directory to /app
WORKDIR /opt/bw/.dist/deb

RUN apt-get update
RUN apt-get -y install dput devscripts dh-make golang dput

CMD /bin/bash build.sh eoan
