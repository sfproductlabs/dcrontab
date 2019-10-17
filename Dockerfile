
FROM golang:1.12
MAINTAINER Andrew Grosser <dioptre@gmail.com>


EXPOSE 6001
EXPOSE 6002
EXPOSE 6003

RUN apt update \
#     && apt upgrade -y --no-install-recommends \     
     && apt install -y build-essential cmake libjemalloc-dev libjemalloc2 libbz2-dev libsnappy-dev zlib1g-dev liblz4-dev libzstd-dev \
     sudo \
     supervisor 

COPY supervisor.conf /etc/supervisor.conf
COPY dcron.supervisor.conf /etc/supervisor/conf.d/dcron.supervisor.conf

# # installing latest gflags
# RUN cd /tmp && \
#     git clone https://github.com/gflags/gflags.git && \
#     cd gflags && \
#     mkdir build && \
#     cd build && \
#     cmake -DBUILD_SHARED_LIBS=1 -DGFLAGS_INSTALL_SHARED_LIBS=1 .. && \
#     make install && \
#     cd /tmp && \
#     rm -R /tmp/gflags/

# # Install Rocksdb
# RUN cd /tmp && \
#     git clone https://github.com/facebook/rocksdb.git && \
#     cd rocksdb && \
#     git checkout v6.3.6 && \
#     make shared_lib && \
#     mkdir -p /usr/local/rocksdb/lib && \
#     mkdir /usr/local/rocksdb/include && \
#     cp librocksdb.so* /usr/local/rocksdb/lib && \
#     cp /usr/local/rocksdb/lib/librocksdb.so* /usr/lib/ && \
#     cp -r include /usr/local/rocksdb/ && \
#     cp -r include/* /usr/include/ && \
#     rm -R /tmp/rocksdb/

# #Install Gorocksdb
# RUN CGO_CFLAGS="-I/usr/local/rocksdb/include" \
#     CGO_LDFLAGS="-L/usr/local/rocksdb/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd" \
#     go get github.com/tecbot/gorocksdb

# # RUN go get github.com/tecbot/gorocksdb
ENV ROCKSDB_VER=5.17.2

WORKDIR $HOME/src
RUN git clone https://github.com/lni/dragonboat
WORKDIR $HOME/src/dragonboat
# RUN sed -i 's/ROCKSDB_MAJOR_VER\=.*$/ROCKSDB_MAJOR_VER\=5/ig' Makefile
# RUN sed -i 's/ROCKSDB_MINOR_VER\=.*$/ROCKSDB_MINOR_VER\=17/ig' Makefile
# RUN sed -i 's/ROCKSDB_PATCH_VER\=.*$/ROCKSDB_PATCH_VER\=2/ig' Makefile
RUN make install-rocksdb-ull

WORKDIR /app/dcrontab
ADD . /app/dcrontab
RUN go get
RUN make

#sudo docker build -t dcrontab .
CMD bash dockercmd.sh

