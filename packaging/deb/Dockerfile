FROM debian:9

RUN apt-get update && \
    apt-get install -y ruby ruby-dev rubygems build-essential git

RUN gem install --no-document fpm -v 1.11.0

VOLUME /repo
WORKDIR /repo

ENV VERSION=

CMD ./packaging/deb/build.sh "$VERSION"