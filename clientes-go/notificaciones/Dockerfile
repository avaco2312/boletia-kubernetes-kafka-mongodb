FROM alpine as alpine
RUN apk add -U --no-cache ca-certificates
FROM scratch
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY notificaciones /
COPY .aws /.aws
ENV HOME=/
CMD ["/notificaciones"]