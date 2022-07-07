ARG BUILD_FROM
FROM $BUILD_FROM
ARG SUPERVISOR_TOKEN
WORKDIR /usr/src/app
COPY . .
RUN chmod a+x ./jablotron.run
CMD ["./jablotron.run", $(bashio::services mqtt "password")] --v
