FROM golang:1.22-alpine

# This is necessary for China devps
RUN go env -w GOPROXY=https://goproxy.cn,direct
# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

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

RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.22.2

# Set the Current Working Directory inside the container
WORKDIR /var/www/example

COPY . .

RUN go mod tidy

ENTRYPOINT ["make","ui"]
EXPOSE 9090
