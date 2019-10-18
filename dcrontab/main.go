//===----------- dcrontab - distributed crontab written in go  -------------===
//
//  Copyright (c) 2018 Andrew Grosser. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"net"
	"errors"
	"sort"
	"os/exec"
	"strconv"
	"regexp"

	"github.com/lni/dragonboat/v3"
	//"github.com/lni/dragonboat/plugin/rpc"
	"github.com/lni/dragonboat/v3/config"
	"github.com/lni/dragonboat/v3/logger"
	"github.com/lni/goutils/syncutil"
	"github.com/lni/goutils/logutil"
)

type RequestType uint64

const (
	PUT RequestType = iota
	GET
	DELETE	
)

const (
	ROOT_KEY string = "__dcron"
)

var (
	plog = logger.GetLogger("dcrontab")
)

// NodeID returns a human friendly form of NodeID for logging purposes.
func NodeID(nodeID uint64) string {
	return logutil.NodeID(nodeID)
}

// ClusterID returns a human friendly form of ClusterID for logging purposes.
func ClusterID(clusterID uint64) string {
	return logutil.ClusterID(clusterID)
}

var dn = logutil.DescribeNode


type Service struct {
	Service  string
	Hosts    []string
	CACert   string
	Cert     string
	Key      string
	Secure   bool
	Critical bool
	Context  string
	Format 	 string
}

//* = everything, minute (0-59), hour (0-23, 0 = midnight), day (1-31), month (1-12), weekday (0-6, 0 = Sunday). 
// Ex. every 10 minutes */10
type Cron struct {
	Minute string
	Hour string
	Day string
	Month string
	Weekday string
	Once bool
}

type Command struct {
	Type string
	Exec string
	Args string
	Timeout time.Duration
	Secure bool
	Retry int
	Critical bool
	Cron Cron
	Comment string
}

type Configuration struct {
	ClusterID 				 uint64
	NodeID					 int
	Address				 	 string
	Addresses                []string 
	UseTLS              	 bool
	TLSCACert                string
	TLSCert                  string
	TLSKey                   string
	Commands                 []Command
	Debug                    bool
	NatsService				 Service
}

func checkCron(cron *Cron) bool {
	//Let's check!
	//* = everything, minute (0-59), hour (0-23, 0 = midnight), day (1-31), month (1-12), weekday (0-6, 0 = Sunday). 
	// Ex. every 10 minutes */10
	every := regexp.MustCompile(`\*\/([0-9]+)`)
	t := time.Now().UTC()

	//////////////////////////////////////// Minutes
	min := t.Minute()
	minFound := false
	if cron.Minute == "*" || cron.Minute == "" {
		minFound = true
	} else {
		for _, v := range strings.Split(cron.Minute,",") {
			if v == "*" {
				minFound = true
				break
			}
			if m,err := strconv.Atoi(v); err == nil {
				if min == m {
					minFound = true
					break
				}
			}
			if div := every.FindStringSubmatch(v); len(div) > 1 {
				if m,err := strconv.Atoi(div[1]); err == nil {
					for check := 0; check < 60; check+=m {					
						if min == check || m < 1 {
							minFound = true
							break
						}
					}
				}	
			}
		}
	}
	if (!minFound) {
		return false
	}

	//////////////////////////////////////// Hours
	hr := t.Hour()
	hrFound := false
	if cron.Hour == "*" || cron.Hour == "" {
		hrFound = true
	} else {
		for _, v := range strings.Split(cron.Hour,",") {
			if v == "*" {
				hrFound = true
				break
			}
			if h,err := strconv.Atoi(v); err == nil {
				if hr == h {
					hrFound = true
					break
				}
			}
			if div := every.FindStringSubmatch(v); len(div) > 1 {
				if h,err := strconv.Atoi(div[1]); err == nil {
					for check := 0; check < 24; check+=h {					
						if hr == check || h < 1 {
							hrFound = true
							break
						}
					}
				}	
			}
		}
	}
	if (!hrFound) {
		return false
	}

	//////////////////////////////////////// Day
	day := t.Day()
	dayFound := false
	if cron.Day == "*" || cron.Day == "" {
		dayFound = true
	} else {
		for _, v := range strings.Split(cron.Day,",") {
			if v == "*" {
				dayFound = true
				break
			}
			if d,err := strconv.Atoi(v); err == nil {
				if day == d {
					dayFound = true
					break
				}
			}
			if div := every.FindStringSubmatch(v); len(div) > 1 {
				if d,err := strconv.Atoi(div[1]); err == nil {
					for check := 0; check < 60; check+=d {					
						if day == check || d < 1 {
							dayFound = true
							break
						}
					}
				}	
			}
		}
	}
	if (!dayFound) {
		return false
	}


	//////////////////////////////////////// Month
	mon := int(t.Month())
	monFound := false
	if cron.Month == "*" || cron.Month == "" {
		monFound = true
	} else {
		for _, v := range strings.Split(cron.Month,",") {
			if v == "*" {
				monFound = true
				break
			}
			if mo,err := strconv.Atoi(v); err == nil {
				if mon == mo {
					monFound = true
					break
				}
			}
			if div := every.FindStringSubmatch(v); len(div) > 1 {
				if mo,err := strconv.Atoi(div[1]); err == nil {
					for check := 1; check < 13; check+=mo {					
						if mon == check || mo < 1 {
							monFound = true
							break
						}
					}
				}	
			}
		}
	}
	if (!monFound) {
		return false
	}

	//////////////////////////////////////// Weekday
	wday := int(t.Weekday())
	wdayFound := false
	if cron.Weekday == "*" || cron.Weekday == "" {
		wdayFound = true
	} else {
		for _, v := range strings.Split(cron.Weekday,",") {
			if v == "*" {
				wdayFound = true
				break
			}
			if wd,err := strconv.Atoi(v); err == nil {
				if wday == wd {
					wdayFound = true
					break
				}
			}
			if div := every.FindStringSubmatch(v); len(div) > 1 {
				if wd,err := strconv.Atoi(div[1]); err == nil {
					for check := 0; check < 7; check+=wd {					
						if wday == check || wd < 1 {
							wdayFound = true
							break
						}
					}
				}	
			}
		}
	}
	if (!wdayFound) {
		return false
	}

	//TODO: Seconds
	//fmt.Println(t.Second())


	return true
}

func parseCommand(msg string) (RequestType, string, string, bool) {
	parts := strings.Split(strings.TrimSpace(msg), " ")
	if len(parts) == 0 || (parts[0] != "put" && parts[0] != "get") {
		return PUT, "", "", false
	}
	if parts[0] == "put" {
		if len(parts) != 3 {
			return PUT, "", "", false
		}
		return PUT, parts[1], parts[2], true
	}
	if len(parts) != 2 {
		return GET, "", "", false
	}
	return GET, parts[1], "", true
}

func printUsage() {
	fmt.Fprintf(os.Stdout, "*** Usage *** \n")
	fmt.Fprintf(os.Stdout, "put key value\n")
	fmt.Fprintf(os.Stdout, "get key\n")
	fmt.Fprintf(os.Stdout, "exit\n")
}


func publicIPs() ([]string, error) {
	ips := make([]string, 0)
	ifaces, err := net.Interfaces()
	if err != nil {
		return []string{}, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return []string{}, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			ips = append(ips,ip.String())
		}
	}
	if (len(ips) > 0) {
		return ips, nil
	}
	return []string{}, errors.New("are you connected to the network?")
}


func main() {
	fmt.Println("\n\n//////////////////////////////////////////////////////////////")
	fmt.Println("Dcrontab.")
	fmt.Println("Distributed crontab using raft consensus")
	fmt.Println("https://github.com/dioptre/dcrontab")
	fmt.Println("(c) Copyright 2019 SF Product Labs LLC.")
	fmt.Println("Use of this software is subject to the LICENSE agreement.")
	fmt.Println("//////////////////////////////////////////////////////////////\n\n")
	
	//////////////////////////////////////// SETUP ARGS
	configFile := flag.String("config", "config.json", "Config file to use")
	nodeID := flag.Int("nodeid", 0, "NodeID to use")
	addr := flag.String("addr", "", "Nodehost address")
	join := flag.Bool("join", false, "Joining a new node")
	flag.Parse()

	//////////////////////////////////////// LOAD CONFIG
	fmt.Println("Configuration file: ", *configFile)
	file, _ := os.Open(*configFile)
	defer file.Close()
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("[ERROR] Configuration error:", err)
		os.Exit(1)
	}
	
	//////////////////////////////////////// SETUP NETWORK
	// https://github.com/golang/go/issues/17393
	if runtime.GOOS == "darwin" {
		signal.Ignore(syscall.Signal(0xd))
	}
	ips, err := publicIPs()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Public IPs: %s\n", ips)	
	peers := make(map[uint64]string)
	if !*join {
		for idx, v := range configuration.Addresses {
			peers[uint64(idx+1)] = v
			if *nodeID == 0 {
				for _, ip := range ips {
					if strings.HasPrefix(v, ip) {
						*nodeID = idx + 1;
						break
					}
				}
			}
		}
	}
	
	if len(*addr) == 0 && (*nodeID == 0 || *nodeID > len(configuration.Addresses)) {
		fmt.Fprintf(os.Stderr, "[ERROR] nodeid must be one of the addresses specified in config or a public ip address must match an address\n")
		os.Exit(1)
	}

	var nodeAddr string
	if len(*addr) != 0 {
		nodeAddr = *addr
	} else {
		nodeAddr = peers[uint64(*nodeID)]
	}
	fmt.Fprintf(os.Stdout, "Node address: %s, Node ID: %d\n", nodeAddr, *nodeID)
	
	//////////////////////////////////////// Setup Loggers
	//logger.SetLoggerFactory()
	logger.GetLogger("raft").SetLevel(logger.ERROR)
	logger.GetLogger("rsm").SetLevel(logger.WARNING)
	logger.GetLogger("transport").SetLevel(logger.WARNING)
	logger.GetLogger("grpc").SetLevel(logger.WARNING)

	//////////////////////////////////////// Setup NATS
	gonats := NatsService{
		Configuration: &configuration.NatsService,
		AppConfig:     &configuration,
	}
	
	err = gonats.connect()	 
	
	if err != nil && &configuration.NatsService != nil {
		plog.Panicf("%s [CRITICAL] Could not connect to NATS Cluster.\n, %v", dn(configuration.ClusterID, uint64(*nodeID)), err)
	}

	//////////////////////////////////////// Setup RAFT
	rc := config.Config{
		NodeID:             uint64(*nodeID),
		ClusterID:          configuration.ClusterID,
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		CheckQuorum:        true,
		SnapshotEntries:    10,
		CompactionOverhead: 5,
	}
	datadir := filepath.Join(
		"dcrontab-data",
		"nodes",
		fmt.Sprintf("node%d", *nodeID))
	nhc := config.NodeHostConfig{
		WALDir:         datadir,
		NodeHostDir:    datadir,
		RTTMillisecond: 200,
		RaftAddress:    nodeAddr,
		//RaftRPCFactory: rpc.NewRaftGRPC,
		MutualTLS: configuration.UseTLS,
		CAFile: configuration.TLSCACert,
		CertFile: configuration.TLSCert,
		KeyFile: configuration.TLSKey,
	}
	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		panic(err)
	}
	if err := nh.StartOnDiskCluster(peers, *join, NewCommander, rc); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] failed to add cluster, %v\n", err)
		os.Exit(1)
	}

	//////////////////////////////////////// RUN SERVICES
	leaderStopper := syncutil.NewStopper()
	raftStopper := syncutil.NewStopper()
	consoleStopper := syncutil.NewStopper()
	ch := make(chan string, 16)
	consoleStopper.RunWorker(func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			s, err := reader.ReadString('\n')
			if err != nil {
				close(ch)
				return
			}
			if s == "exit\n" {
				raftStopper.Stop()
				nh.Stop()
				return
			}
			ch <- s
		}
	})

	leaderStopper.RunWorker(func() {
		cs := nh.GetNoOPSession(configuration.ClusterID)
		// this goroutine makes a linearizable read every 60 seconds
		// then runs the commands required
		nextTime :=  time.Now().Truncate(time.Minute)
		time.Sleep(time.Until(nextTime.Add(1*time.Second)))
		nextTime =  nextTime.Add(1*time.Minute)
		ticker := time.NewTicker(60 * time.Second)
		for {
			select {
			case <-ticker.C:
				leader, _, _ := nh.GetLeaderID(configuration.ClusterID)
				if  leader == uint64(*nodeID) {
					action := &Action{
						Action: "SCAN",
						Key: ROOT_KEY,
					}
					data, err := json.Marshal(action)
					if err != nil {
						panic(err)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 58*time.Second)
					result, err := nh.SyncRead(ctx, configuration.ClusterID, data)
					if err == nil {											
						items := make(map[string]string)
						json.Unmarshal(result.([]byte), &items)
						if len(items) == 0 {
							plog.Infof("%s [SETTING UP] : Running\n", dn(configuration.ClusterID, uint64(*nodeID)))							
							//First time we've run
							//Add the root key 
							action := &Action{
								Action: "PUT",
								Key: ROOT_KEY,
								Val: "OK",
							}
							data, err := json.Marshal(action)
							if err != nil {
								panic(err)
							}
							_, err = nh.SyncPropose(ctx, cs, data)
							if err != nil {
								plog.Infof("%s [ERROR] SyncPropose returned error %v\n", dn(configuration.ClusterID, uint64(*nodeID)), err)
							}
							//and populate commands
							for idx, v := range configuration.Commands {
								action.Key = fmt.Sprintf("%s::%d", ROOT_KEY, idx)
								if data, err = json.Marshal(v); err == nil {
									action.Val = string(data)
									if a, e := json.Marshal(action); e == nil {
										_, e = nh.SyncPropose(ctx, cs, a)
									}
								}
							}
							plog.Infof("%s [SETTING UP] : Complete\n", dn(configuration.ClusterID, uint64(*nodeID)))							
						} else {
							var keys []string
							for k := range items {
								keys = append(keys, k)
							}
							sort.Strings(keys)
							for _, k := range keys { 
								if (k == ROOT_KEY) {
									continue;
								}								
								cmd := &Command{}

								plog.Infof("%s [EXECUTION INIT] job %s\n", dn(configuration.ClusterID, uint64(*nodeID)), k)							
								if err:= json.Unmarshal([]byte(items[k]), &cmd); err == nil && cmd.Type != "" && cmd.Exec != "" {
									if &cmd.Cron != nil && !checkCron(&cmd.Cron) {
										plog.Infof("%s [EXECUTION ABORTED] Not time for job %s\n", dn(configuration.ClusterID, uint64(*nodeID)), k)							
										continue
									}
									switch cmd.Type {
									case "nats":
										gonats.publish(cmd.Exec, cmd.Args)
										plog.Infof("%s [EXECUTION COMPLETE] NATS job %s, Input:\n%s\n", dn(configuration.ClusterID, uint64(*nodeID)), k, cmd)							
									case "shell":
										//TODO: move to a go subroutine
										var run *exec.Cmd
										if len(cmd.Args) > 0 {
											run = exec.Command(cmd.Exec, cmd.Args)
										} else {
											run = exec.Command(cmd.Exec)										
										}
										out, err := run.CombinedOutput()
										if err == nil {
											plog.Infof("%s [EXECUTION COMPLETE] SHELL job %s, Input:\n%s\nOutput:\n%s\n", dn(configuration.ClusterID, uint64(*nodeID)), k, cmd, string(out))							
										} else {
											plog.Infof("%s [ERROR] Execution failed for job %s, Input:\n%s\nOutput:\n%s\nError:\n%v\n", dn(configuration.ClusterID, uint64(*nodeID)), k, cmd, string(out), err)							
										}
										
									}
								} else {
									plog.Infof("%s [ERROR] Parsing job %s: %v\n", dn(configuration.ClusterID, uint64(*nodeID)), k, err)	
								}
							}
						}
					} else {
						//TODO: Remove						
						//panic(err)
					}
					nextTime = nextTime.Add(time.Minute)
					if until := time.Until(nextTime); until > 0 {
						time.Sleep(until)
						plog.Infof("%s %s [TIMER]\n", time.Now(), until)
						ticker = time.NewTicker(60 * time.Second)
					} else {
						nextTime = time.Now().Truncate(time.Minute)
					}
					cancel()
				} else {
					plog.Infof("%s [INFO] Deferring command to node: %d\n", dn(configuration.ClusterID, uint64(*nodeID)), leader)	
				}				
			case <-leaderStopper.ShouldStop():
				return
			}			
		}
	})

	raftStopper.RunWorker(func() {
		cs := nh.GetNoOPSession(configuration.ClusterID)
		for {
			select {
			case v, ok := <-ch:
				if !ok {
					return
				}
				msg := strings.Replace(v, "\n", "", 1)
				// input message must be in the following formats -
				// put key value
				// get key
				rt, key, val, ok := parseCommand(msg)
				if !ok {
					fmt.Fprintf(os.Stderr, "invalid input\n")
					printUsage()
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				if rt == PUT {
					action := &Action{
						Action: "PUT",
						Key: key,
						Val: val,
					}
					data, err := json.Marshal(action)
					if err != nil {
						panic(err)
					}
					_, err = nh.SyncPropose(ctx, cs, data)
					if err != nil {
						plog.Infof("%s [ERROR] SyncPropose returned error %v\n", dn(configuration.ClusterID, uint64(*nodeID)), err)							
					}
				} else {
					result, err := nh.SyncRead(ctx, configuration.ClusterID, []byte(key))
					if err != nil {
						plog.Infof("%s [ERROR] SyncRead returned error %v\n", dn(configuration.ClusterID, uint64(*nodeID)), err)	
					} else {
						plog.Infof("%s [INFO] Successful query key: %s, result: %s \n", dn(configuration.ClusterID, uint64(*nodeID)), key, result)	
					}
				}
				cancel()
			case <-raftStopper.ShouldStop():
				return
			}
		}
	})
	raftStopper.Wait()
}
