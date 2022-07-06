ARG BUILD_FROM=golang:1.18
FROM $BUILD_FROM
WORKDIR /usr/src/app
COPY . .
RUN chmod a+x /jablotron.run
CMD ["/jablotron.run"] --v