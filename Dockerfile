# 
# This dockerfile was written special for launching the Portal_Cinema BOT
# Visit our site by url [ https://onix-systems.com ] to read more about our team
#

FROM ubuntu:xenial

ENV GO_VERSION="1.4.2" \
    GOROOT="/usr/local/go" 
ENV GOPATH="${GOROOT}/src" \
    PATH="${GOROOT}/bin:${PATH}" \
    PROJECT_FOLDER="/opt/project"    

# Setting our native timezone
RUN ln -s -f /usr/share/zoneinfo/Europe/Kiev /etc/localtime

RUN sed -i "s/http:\/\/archive.ubuntu/http:\/\/ua.archive.ubuntu/g" /etc/apt/sources.list

RUN apt-get update

RUN apt-get install -y \
        wget \
        git

RUN wget https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz -O /tmp/go.tar.gz

RUN tar -xvf /tmp/go.tar.gz

RUN mv go /usr/local

# Install nessesary Go-packages
RUN  go get \
        gopkg.in/telegram-bot-api.v4 \
        gopkg.in/mgo.v2 \
        gopkg.in/mgo.v2/bson \
        gopkg.in/robfig/cron.v2

# Copying the project inside the container
RUN mkdir -p ${PROJECT_FOLDER}

WORKDIR ${PROJECT_FOLDER}

COPY * ./

RUN go build main.go

CMD ["./main"]
