FROM ubuntu:18.10

# Set the working directory to /app
WORKDIR /opt/bw/.dist/deb

RUN apt-get update
RUN apt-get -y install dput devscripts dh-make golang
RUN ["/bin/bash", "build.sh", "bionic"]
RUN ["/bin/bash", "build.sh", "eoan"]
CMD /bin/bash
