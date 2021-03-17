FROM registry.cn-hangzhou.aliyuncs.com/leistudy/golang:1.16

ADD ./ /usr/local/go/src/gameServer-demo

WORKDIR "/usr/local/go/src/gameServer-demo"

ENV GOPROXY https://goproxy.io

ENV GO111MODULE on

RUN ["go", "mod", "tidy"]

RUN ["go", "build", "-o", "/main", "src/main.go"]

CMD ["/main"]
