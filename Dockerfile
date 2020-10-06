FROM amazonlinux AS build
RUN yum -y update && yum -y install tar gzip gcc
RUN curl -o go1.14.3.linux-amd64.tar.gz https://dl.google.com/go/go1.14.3.linux-amd64.tar.gz -s
RUN tar -C /usr/local -xzf go1.14.3.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
WORKDIR /src/aws-secrets-manager
COPY ./go.mod ./go.sum ./
RUN go mod download
COPY . ./
ARG APP=initcontainer
RUN go test ./${APP}
RUN go build -o /app -v ./${APP}

FROM amazonlinux:latest
RUN yum -y update && yum install -y ca-certificates && rm -rf /var/cache/yum/*
COPY --from=build /app /.
ENTRYPOINT ["/app"]
