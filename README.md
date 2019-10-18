# Distributed crontab (dcrontab)

* Supports Running a single crontab across a cluster of machines with a resolution down to a minute
* Use this for critical service operations like updating auction items at the end of an auction.

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

See the config.json file for all the options.

#### Requirements
**Go > Version 1.12**

#### Optional Requirements

NATS

#### Building

```
make
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
