ARG BUILD_FROM
FROM $BUILD_FROM
WORKDIR /usr/src/app
COPY run.sh /
COPY jablotron.run /
RUN chmod a+x /run.sh
RUN chmod a+x /jablotron.run
CMD ["./run.sh"] --v
