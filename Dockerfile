ARG BUILD_FROM
#FROM $BUILD_FROM
FROM golang:1.18
WORKDIR /data
#COPY jablotron.run /
#RUN chmod a+x /jablotron.run
#CMD ["/jablotron.run"] --v
copy jablotron.go / 
copy go.mod / 
CMD ["go run ."] --v