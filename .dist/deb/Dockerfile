FROM ubuntu:18.10

# Set the working directory to /app
WORKDIR /opt/bw/.dist/deb

RUN apt-get update
RUN apt-get -y install dput devscripts dh-make golang dput

CMD /bin/bash build.sh bionic disco eoan
