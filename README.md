## Distributed crontab (dcrontab)

#### Requirements
**Go > Version 1.12**

WORK IN PROGRESS

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
You can type in a message in one of the two following formats - 
```
put key value
```
or 
```
get key
```
