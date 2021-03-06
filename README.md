# Distributed crontab (dcrontab)

* Supports Running a single crontab across a cluster of machines with a resolution down to a minute
* Use this for critical service operations like updating auction items at the end of an auction.


## Getting the package from Github docker registry

https://github.com/dioptre/dcrontab/packages/48583

## Running on AWS
* Use the production branch https://github.com/dioptre/dcrontab/tree/production
* You can use docker or build manually if you wish (see the commented lines in ./Dockerfile - assumes a debian/Ubuntu distribution)
* Setup dcrontab1 and more on your private dns (see the config file) names will resolve
* Generate the certs per the ./gencerts.sh (the first time generate a CA Ex. ```./gencerts 1 ca```)
* Update the config file with your key/node (use dcrontab[0-9]*)
* **The example requires a nats service to be setup, but you can disable it if you wish.**

## Running on Docker
From the directory:

```sudo docker build -t dcrontab .```

### Docker Compose
Add this to your docker-compose.yml (Version 3). You must add >1 machines to cluster.

Example:
```
version: '3'
services:
  nats:
    build: nats:latest
    ports:
      - "4222:4222"
      - "6222:6222"
      - "8222:8222"
    networks:
      - default
  dcron1:
    image: dcrontab:latest
    command: sh -c '/app/dcrontab/wait-for localhost:4222 -t 300 -- sleep 3 && /app/dcrontab/dockercmd.sh'
    depends_on:
      - "nats"
    expose:
      - "6001"
    ports:
      - "6001:6001"
    network_mode: host  
    environment:
      - "NODEID=1"   
dcron2:
    image: dcrontab:latest
    command: sh -c '/app/dcrontab/wait-for localhost:4222 -t 300 -- sleep 3 && /app/dcrontab/dockercmd.sh'
    depends_on:
      - "nats"
    expose:
      - "6002"
    ports:
      - "6002:6002"
    network_mode: host  
    environment:
      - "NODEID=2"   
dcron3:
    image: dcrontab:latest
    command: sh -c '/app/dcrontab/wait-for localhost:4222 -t 300 -- sleep 3 && /app/dcrontab/dockercmd.sh'
    depends_on:
      - "nats"
    expose:
      - "6003"
    ports:
      - "6003:6003"
    network_mode: host  
    environment:
      - "NODEID=3"         
```

Then run:
```sudo docker-compose up```

## Running from source

### Setup
Follow the instructions for building inside the 
```Dockerfile```

See the **config.json** file for all the options.

#### Requirements / Dependencies
**Go > Version 1.12**
RocksDB (try something like brew install rocksdb)

#### Optional Requirements

NATS - https://nats.io

#### Building from source

```
sudo apt update \
     && sudo apt install -y build-essential cmake libjemalloc-dev libbz2-dev libsnappy-dev zlib1g-dev liblz4-dev libzstd-dev \
     sudo \
     supervisor \
     netcat

sudo apt install git
sudo apt upgrade

wget https://dl.google.com/go/go1.13.4.linux-amd64.tar.gz
tar -xvf go1.13.4.linux-amd64.tar.gz
sudo mv go /usr/local
mkdir projects
cd projects/
mkdir go
#vi ~/.bashrc 

## Add to .bashrc
echo "export GOROOT=/usr/local/go" >> ~/.bashrc
echo "export GOPATH=$HOME/projects/go" >> ~/.bashrc
echo "export PATH=$HOME/projects/go/bin:/usr/local/go/bin:$PATH" >> ~/.bashrc

# installing latest gflags
cd /tmp && \
     git clone https://github.com/gflags/gflags.git && \
     cd gflags && \
     mkdir build && \
     cd build && \
     cmake -DBUILD_SHARED_LIBS=1 -DGFLAGS_INSTALL_SHARED_LIBS=1 .. && \
     sudo make install && \
     cd /tmp && \
     rm -R /tmp/gflags/

# Install Rocksdb
cd /tmp && \
     git clone https://github.com/facebook/rocksdb.git && \
     cd rocksdb && \
     git checkout v6.3.6 && \
     make shared_lib && \
     sudo mkdir -p /usr/local/rocksdb/lib && \
     sudo mkdir /usr/local/rocksdb/include && \
     sudo cp librocksdb.so* /usr/local/rocksdb/lib && \
     sudo cp /usr/local/rocksdb/lib/librocksdb.so* /usr/lib/ && \
     sudo cp -r include /usr/local/rocksdb/ && \
     sudo cp -r include/* /usr/include/ && \
     rm -R /tmp/rocksdb/

#Install Gorocksdb
CGO_CFLAGS="-I/usr/local/rocksdb/include" \
     CGO_LDFLAGS="-L/usr/local/rocksdb/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd" \
     go get github.com/tecbot/gorocksdb


cd ~/projects
git clone https://github.com/dioptre/dcrontab
cd dcrontab
make

sudo mkdir /app
sudo chown admin:admin /app
ln -s /home/admin/projects/dcrontab /app/dcrontab

sudo ln /home/admin/projects/dcrontab/supervisor.conf /etc/supervisor.conf
sudo ln /home/admin/projects/dcrontab/dcron.supervisor.conf /etc/supervisor/conf.d/dcron.supervisor.conf


sudo systemctl enable supervisor.service

##UPDATE THE CONFIG FILE

## Change hostname on amazon jessie 
#sudo hostnamectl set-hostname dcrontab1
#sudo reboot

```

then 

```
make
```

or on mac I need to

```
IPHONEOS_DEPLOYMENT_TARGET= SDKROOT= make
```

#### Running

```
./dcrontab -addr localhost:6001 -nodeid 1
#... add more nodes to more machines
```

### Adding items to crontab
You can type in a cronjob directly into the console - but better to manage the jobs with the config.json.
```
put key value
```
Ex.
```
put __dcron::99 {"Type":"shell","Exec":"ls"}
```
or 
```
get key
```

### TODO

- [ ] Run once (equivalent to @reboot)
- [ ] Resolution down to seconds (minutes atm)
- [ ] DELETE method
- [ ] HTTPS & Auth Support


#### Credits

Andrew Grosser - https://sfpl.io

Lei Ni - https://github.com/lni/dragonboat
