
FROM golang:1.12
MAINTAINER Andrew Grosser <dioptre@gmail.com>


EXPOSE 6001
EXPOSE 6002
EXPOSE 6003

RUN apt update \
     && apt install -y build-essential cmake libjemalloc-dev libbz2-dev libsnappy-dev zlib1g-dev liblz4-dev libzstd-dev \
     sudo \
     supervisor \
     netcat

############
## Physical instructions:
############
# apt install git
# apt upgrade

## Get good go build 
## https://golang.org/dl/

# wget https://dl.google.com/go/go1.13.4.linux-amd64.tar.gz
#tar -xvf go1.13.3.linux-amd64.tar.gz
#sudo mv go /usr/local
#mkdir projects
#cd projects/
#mkdir go
#vi ~/.bashrc 

## Add to .bashrc
#echo "export GOROOT=/usr/local/go" >> ~/.bashrc
#echo "export GOPATH=$HOME/projects/go" >> ~/.bashrc
#echo "export PATH=$HOME/projects/go/bin:/usr/local/go/bin:$PATH" >> ~/.bashrc

# cd ~/projects
# git clone https://github.com/lni/dragonboat
# cd dragonboat
# ROCKSDB_VER=5.17.2 make install-rocksdb-ull

# cd ~/projects
# git clone https://github.com/dioptre/dcrontab
# cd dcrontab/dcrontab
# go get
# cd ..
# make

#sudo mkdir /app
#sudo chown admin:admin /app
#ln -s /home/admin/projects/dcrontab /app/dcrontab

#sudo ln /home/admin/projects/dcrontab/supervisor.conf /etc/supervisor.conf
#sudo ln /home/admin/projects/dcrontab/dcron.supervisor.conf /etc/supervisor/conf.d/dcron.supervisor.conf

##UPDATE THE CONFIG FILE

## Change hostname on amazon jessie 
#sudo hostnamectl set-hostname dcrontab1
#sudo reboot

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
RUN rm -rf dcrontab-data
RUN go get
RUN make

#sudo docker build -t dcrontab .
CMD bash dockercmd.sh

