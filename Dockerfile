ARG BUILD_FROM=golang:1.18
#FROM $BUILD_FROM
WORKDIR /data
#COPY jablotron.run /
#RUN chmod a+x /jablotron.run
#CMD ["/jablotron.run"] --v
copy jablotron.go / 
copy go.mod / 
CMD ["go run ."] --v