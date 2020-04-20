FROM golang AS builder
WORKDIR /go/src/app
COPY . .
ENV CGO_ENABLED=0
RUN go get -d -v 
RUN go test && go build -o /go/bin/app

FROM alpine:3.9
ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT
RUN apk add --no-cache tzdata
ENV TZ Europe/Moscow
RUN addgroup -S app && adduser -S -G app app && mkdir /opt/app && chown app:app /opt/app
USER app
COPY --chown=app:app --from=builder /go/bin/app /opt/app/alertmanager-megafon-sms
WORKDIR /opt/app/
EXPOSE 9097
ENTRYPOINT ["/opt/app/alertmanager-megafon-sms"]
