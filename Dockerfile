ARG BUILD_FROM
FROM $BUILD_FROM
WORKDIR /data
COPY jablotron.run /
RUN chmod a+x /jablotron.run
CMD ["/jablotron.run"] --v