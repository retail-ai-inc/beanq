FROM golang:1.23-alpine

ARG USE_CHINA_PROXY
ENV USE_CHINA_PROXY=${USE_CHINA_PROXY}

RUN set -eux; \
  if [ "${USE_CHINA_PROXY}" = "true" ]; then \
  echo "Using China Proxy"; \
  go env -w GOPROXY=https://goproxy.cn,direct; \
  sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories; \
  else \
  echo "Using default Golang Proxy"; \
  go env -w GOPROXY=https://proxy.golang.org,direct; \
  fi

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
RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.22.2

# Set the Current Working Directory inside the container
WORKDIR /var/www/example

COPY . .

RUN go mod tidy

ENTRYPOINT ["make","ui"]
EXPOSE 9090