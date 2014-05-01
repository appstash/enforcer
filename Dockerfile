FROM pandrew/ubuntu-lts

MAINTAINER Paul Andrew Liljenberg "letters@paulnotcom.se"

RUN  apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -yq \
    dpkg-sig \
    build-essential \
    mercurial \
    ruby1.9.1 \
    ruby1.9.1-dev \
    s3cmd \
    --no-install-recommends

# Install Go
RUN curl -s https://go.googlecode.com/files/go1.2.1.src.tar.gz | tar -v -C /usr/local -xz
ENV PATH    /usr/local/go/bin:/go/bin:$PATH
ENV GOPATH  /go:/go/src/github.com/dotcloud/docker/vendor
RUN cd /usr/local/go/src && ./make.bash --no-clean 2>&1

# Add gox
RUN go get github.com/mitchellh/gox
RUN /go/bin/gox -build-toolchain

# Add some dependencies
RUN go get github.com/dancannon/gorethink
RUN go get github.com/gorilla/mux
RUN go get github.com/hashicorp/serf
RUN go get github.com/dotcloud/docker/pkg/mflag

RUN gem install --no-rdoc --no-ri fpm --version 1.0.2

RUN /bin/echo -e '[default]\naccess_key=$AWS_ACCESS_KEY\nsecret_key=$AWS_SECRET_KEY' > /.s3cfg
RUN echo $AWS_ACCESS_KEY

VOLUME /opt/enforcer
WORKDIR /go/src/github.com/appstash/enforcer

# Wrap all commands in the "docker-in-docker" script to allow nested containers
ENTRYPOINT    ["hack/dind"]

# Upload docker source
ADD   .   /go/src/github.com/appstash/enforcer
