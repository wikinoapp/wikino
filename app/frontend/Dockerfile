FROM node:12.13.0-alpine

LABEL maintainer="https://github.com/nonoto" \
      description="A note app."

RUN apk update && \
    apk add -t \
        git

ENV PATH=./node_modules/.bin/:$PATH \
    NODE_ENV=development

WORKDIR /nonoto/

COPY package.json yarn.lock tsconfig.json webpack.config.js ./
RUN yarn install && \
    yarn cache clean

EXPOSE 8080
