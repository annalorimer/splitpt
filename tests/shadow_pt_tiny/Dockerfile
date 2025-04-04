FROM debian:12-slim as base
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
ENV DEBIAN_FRONTEND=noninteractive
# use NRL 5540 branch package caches
#COPY sources.list /etc/apt/sources.list
#COPY pip.conf /etc/pip.conf
# NOISE parks cluster config
#COPY resolv.conf /etc/resolv.conf
# Define software versions
ENV SHADOW_COMMIT=db817386689f27ae8661f64479d2b9e34c278af0
ENV TGEN_COMMIT=816d68cd3d0ff7d0ec71e8bbbae24ecd6a636117
ENV ONIONTRACE_COMMIT=3696db43288c8a116e8a1cff42a9c698d1d4ab33
ENV TORNETTOOLS_COMMIT=9716a8682483f363e4bc9f9503f6871536e7b846
ENV LYREBIRD_COMMIT=a1bccd9677e4d7a0e596a7cef4e182f9ba99ff4f
ENV WEBTUNNEL_COMMIT=e64b1b3562f3ab50d06141ecd513a21ec74fe8c6
ENV PROTEUS_COMMIT=99751539b78782d4477411786e4df03b68213e5d
# Get basic packages
RUN apt update && apt install -y apt-utils git sudo
RUN apt install -y locales
RUN dpkg-reconfigure locales
RUN dpkg-reconfigure tzdata

# Build proteus
FROM base as build-proteus
WORKDIR /root/src
RUN git clone --depth=1 -b main https://github.com/unblockable/proteus.git
WORKDIR /root/src/proteus
RUN git fetch origin $PROTEUS_COMMIT
RUN git checkout FETCH_HEAD
RUN test $(git rev-parse HEAD) = $PROTEUS_COMMIT
WORKDIR /root/src
RUN apt update && apt install -y \
  clang \
  curl
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs --output rustup-init.sh 
RUN sh rustup-init.sh -y
ENV PATH "/root/.cargo/bin:${PATH}"
RUN cargo install --root /opt/ --path proteus

# Build lyrebird
FROM base as build-lyrebird
RUN apt update && apt install -y \
make \
gcc \
curl
WORKDIR /root/src
RUN curl -LO https://go.dev/dl/go1.22.2.linux-amd64.tar.gz
RUN tar -xzf go1.22.2.linux-amd64.tar.gz
ENV PATH=/root/src/go/bin:$PATH
WORKDIR /root/src
RUN GIT_SSL_NO_VERIFY=1 git clone --depth=1 -b main https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/lyrebird.git
WORKDIR /root/src/lyrebird
RUN git fetch origin $LYREBIRD_COMMIT
RUN git checkout FETCH_HEAD
RUN test $(git rev-parse HEAD) = $LYREBIRD_COMMIT
RUN sed -i 's/CGO_ENABLED=0/CGO_ENABLED=1/g' Makefile
RUN make
RUN mkdir -p /opt/bin && cp lyrebird /opt/bin/.

# Build webtunnel
FROM base as build-webtunnel
RUN apt update && apt install -y \
make \
gcc \
curl
WORKDIR /root/src
RUN curl -LO https://go.dev/dl/go1.22.2.linux-amd64.tar.gz
RUN tar -xzf go1.22.2.linux-amd64.tar.gz
ENV PATH=/root/src/go/bin:$PATH
RUN GIT_SSL_NO_VERIFY=1 git clone --depth=1 -b main https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/webtunnel.git
WORKDIR /root/src/webtunnel
RUN git fetch origin $WEBTUNNEL_COMMIT
RUN git checkout FETCH_HEAD
RUN test $(git rev-parse HEAD) = $WEBTUNNEL_COMMIT
WORKDIR /root/src/webtunnel/release
RUN sed -i 's/CGO_ENABLED=0/CGO_ENABLED=1/g' build.sh
RUN bash build.sh
RUN mkdir -p /opt/bin && cp build/-/client /opt/bin/webtunnel-client && cp build/-/server /opt/bin/webtunnel-server

# Build shadow
FROM base as build-shadow
WORKDIR /root/src
RUN git clone --depth=1 -b main https://github.com/shadow/shadow.git
WORKDIR /root/src/shadow
RUN git fetch origin $SHADOW_COMMIT
RUN git checkout FETCH_HEAD
RUN test $(git rev-parse HEAD) = $SHADOW_COMMIT
ENV CC gcc
ENV CXX g++
ENV CONTAINER debian:12-slim
ENV BUILDTYPE release
ENV RUSTPROFILE minimal
RUN ci/container_scripts/install_deps.sh
RUN ci/container_scripts/install_extra_deps.sh
ENV PATH "/root/.cargo/bin:${PATH}"
RUN ./setup build --jobs `nproc` --prefix /opt --clean
RUN ./setup install

# Build tgen
FROM base as build-tgen
RUN apt-get update && apt-get install -y \
  cmake \
  libglib2.0-dev \
  libigraph-dev
WORKDIR /root/src
RUN git clone --depth=1 -b main https://github.com/shadow/tgen.git
WORKDIR /root/src/tgen
RUN git fetch origin $TGEN_COMMIT
RUN git checkout FETCH_HEAD
RUN test $(git rev-parse HEAD) = $TGEN_COMMIT
WORKDIR /root/src/tgen/build
RUN cmake .. -DCMAKE_INSTALL_PREFIX=/opt
RUN make -j`nproc`
RUN make install

# Build oniontrace
FROM base as build-oniontrace
RUN apt-get update && apt-get install -y \
  cmake \
  libglib2.0-dev
WORKDIR /root/src
RUN git clone --depth=1 -b main https://github.com/shadow/oniontrace.git
WORKDIR /root/src/oniontrace
RUN git fetch origin $ONIONTRACE_COMMIT
RUN git checkout FETCH_HEAD
RUN test $(git rev-parse HEAD) = $ONIONTRACE_COMMIT
WORKDIR /root/src/oniontrace/build
RUN cmake .. -DCMAKE_INSTALL_PREFIX=/opt
RUN make -j`nproc`
RUN make install

# Build tor
FROM base as build-tor
RUN apt-get update && apt-get install -y \
  autoconf \
  automake \
  gcc \
  make \
  libevent-dev \
  openssl \
  libssl-dev \
  zlib1g-dev \
  libpsl-dev \
  libpsl5
WORKDIR /root/src
ARG tor_repo=https://gitlab.torproject.org/tpo/core/tor.git
ARG tor_branch=tor-0.4.8.13
RUN GIT_SSL_NO_VERIFY=1 git clone --depth=1 -b $tor_branch $tor_repo
WORKDIR /root/src/tor
RUN bash autogen.sh
ENV LIBS=-lpsl
RUN ./configure \
  --prefix=/opt \
  --disable-unittests \
  --disable-asciidoc \
  --disable-manpage \
  --disable-html-manual \
  --disable-gcc-hardening \
  --disable-linker-hardening \
  --disable-fragile-hardening \
  --disable-expensive-hardening \
  --disable-seccomp \
  --disable-system-torrc \
  --disable-dependency-tracking \
  --disable-lzma \
  --disable-zstd \
  --enable-pic
RUN make -j`nproc`
RUN make install

FROM base as build-tornettools
RUN apt-get update && apt-get install -y \
  git \
  python3 \
  python3-pip \
  python3-venv
# Set up our python venv
ENV PYVENV=/opt
RUN python3 -m venv $PYVENV
ENV PATH=$PYVENV/bin:$PATH
# tornettools
WORKDIR /opt/src
RUN git clone --depth=1 -b main https://github.com/shadow/tornettools.git
WORKDIR /opt/src/tornettools
RUN git fetch origin $TORNETTOOLS_COMMIT
RUN git checkout FETCH_HEAD
RUN test $(git rev-parse HEAD) = $TORNETTOOLS_COMMIT
RUN pip3 install -r requirements.txt
RUN pip3 install -I .
# tgentools (needed for parsing)
WORKDIR /opt/src
RUN git clone --depth=1 -b main https://github.com/shadow/tgen.git
WORKDIR /opt/src/tgen
RUN git fetch origin $TGEN_COMMIT
RUN git checkout FETCH_HEAD
RUN test $(git rev-parse HEAD) = $TGEN_COMMIT
WORKDIR /opt/src/tgen/tools
RUN pip3 install -r requirements.txt
RUN pip3 install -I .
# oniontracetools (needed for parsing)
WORKDIR /opt/src
RUN git clone --depth=1 -b main https://github.com/shadow/oniontrace.git
WORKDIR /opt/src/oniontrace
RUN git fetch origin $ONIONTRACE_COMMIT
RUN git checkout FETCH_HEAD
RUN test $(git rev-parse HEAD) = $ONIONTRACE_COMMIT
WORKDIR /opt/src/oniontrace/tools
RUN pip3 install -r requirements.txt
RUN pip3 install -I .

FROM base as simulate
WORKDIR /mnt
# runtime deps for retracer to read hdf5 files.
# RUN apt-get update && apt-get install -y \
#   libhdf5-dev \
#   hdf5-filter-plugin \
#   hdf5-filter-plugin-blosc-serial
# python deps for tornettools etc.
RUN apt-get update && apt-get install -y \
  dstat \
  procps \
  xz-utils \
  zstd
# tor runtime deps
RUN apt-get update && apt-get install -y \
  zlib1g-dev \
  libevent-dev \
  openssl \
  libssl-dev \
  libpsl-dev \
  libpsl5
# shadow runtime deps
RUN apt-get update && apt-get install -y \
  libglib2.0-0 \
  libigraph-dev \
#  libprocps-dev
  libproc2-dev
# perf
#RUN apt-get update && apt-get install -y linux-tools-generic
#RUN mkdir -p /root/.local/bin
#RUN ln -s /usr/lib/linux-tools/5.4.0-81-generic/perf /root/.local/bin/perf
#COPY --from=tornettools-generate /root/data/tornet-* /root/data/tornet
RUN echo fs.nr_open = 104857600 >> /etc/sysctl.conf
RUN echo fs.file-max = 104857600 >> /etc/sysctl.conf
RUN echo vm.max_map_count = 1073741824 >> /etc/sysctl.conf
RUN echo kernel.pid_max = 4194300 >> /etc/sysctl.conf
RUN echo kernel.threads-max = 4194300 >> /etc/sysctl.conf
#RUN sysctl -p
# Get the binaries for our sim tools
COPY --from=build-shadow /opt /opt/
COPY --from=build-tgen /opt /opt/
COPY --from=build-tor /opt /opt/
COPY --from=build-oniontrace /opt /opt/
COPY --from=build-proteus /opt /opt/
COPY --from=build-lyrebird /opt /opt/
COPY --from=build-webtunnel /opt /opt/
COPY --from=build-tornettools /opt /opt/
# Setup run env
ENV PYVENV=/opt
ENV PATH=/opt/bin:$PATH
ENV LD_LIBRARY_PATH=/opt/lib
# source for testing
# COPY --from=build-shadow /root/src/shadow /opt/src/shadow
COPY ../../client/client /opt/bin/splitpt-client
COPY ../../server/server /opt/bin/splitpt-server

# Default command runs the simulation
#CMD tornettools simulate -s /opt/bin/shadow tornet
CMD bash run.sh
