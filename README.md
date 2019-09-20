## Distributed crontab (dcrontab)

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
