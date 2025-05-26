FROM golang:1.23-alpine

# This is necessary for China devps
#RUN go env -w GOPROXY=https://goproxy.cn,direct
#RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

ENV GODEBUG httpmuxgo122=1

# The latest alpine images don't have some tools like (`git` and `bash`).
# Adding git, bash and openssh to the image
RUN apk update &&  \
    apk upgrade &&  \
    apk add --no-cache  \
    bash  \
    git  \
    openssh  \
    make  \
    autoconf  \
    gcc  \
    libc-dev  \
    sudo  \
    procps  \
    curl \
    jq

RUN mkdir -p /var/www/example

RUN go install github.com/go-delve/delve/cmd/dlv@v1.24.2

# Set the Current Working Directory inside the container
WORKDIR /var/www/example

EXPOSE 9090 8888

ENTRYPOINT ["make","ui"]
