FROM fedora:27

RUN yum install -y ruby-devel gcc make rpm-build rubygems git

RUN gem install --no-document fpm -v 1.11.0

VOLUME /repo
WORKDIR /repo

ENV VERSION=

CMD ./packaging/rpm/build.sh "$VERSION"