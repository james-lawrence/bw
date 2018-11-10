# Use an official Python runtime as a parent image
FROM ubuntu:18.10

# Set the working directory to /app
WORKDIR /opt/bw/.dist/deb

RUN apt-get update
RUN apt-get -y install dput devscripts dh-make

CMD bash build.sh
