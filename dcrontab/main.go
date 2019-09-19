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
	// "math"
	// "math/rand"
	"net"
	"errors"
	"sort"
	"os/exec"
	"log"

	"github.com/lni/dragonboat/v3"
	"github.com/lni/dragonboat/v3/config"
	"github.com/lni/dragonboat/v3/logger"
	"github.com/lni/goutils/syncutil"
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
// Ex. 
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

func checkCron(cmd *Command) bool {
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
		log.Fatalf("[CRITICAL] Could not connect to NATS Cluster. %s\n", err)		
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
		// nextTime :=  time.Now().Truncate(time.Minute)
		// time.Sleep(time.Until(nextTime))
		ticker := time.NewTicker(3 * time.Second) //TODO: REPLACE WITH 60 seconds
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
							fmt.Println("[SETTING UP] : Running")
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
								fmt.Fprintf(os.Stderr, "SyncPropose returned error %v\n", err)
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
							fmt.Println("[SETTING UP] : Complete")
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
								if err:= json.Unmarshal([]byte(items[k]), &cmd); err == nil && cmd.Type != "" && cmd.Exec != "" {
									if !checkCron(cmd) {
										continue
									}
									switch cmd.Type {
									case "nats":
										gonats.publish(cmd.Exec, cmd.Args)
										fmt.Printf("[EXECUTION COMPLETE] NATS, Input:\n%s\n", cmd)	
									case "shell":
										//TODO: move to a go subroutine
										cmd := exec.Command(cmd.Exec, cmd.Args)
										out, err := cmd.CombinedOutput()
										if err == nil {
											fmt.Printf("[EXECUTION COMPLETE] SHELL, Input:\n%s\nOutput:\n%s\n", cmd, string(out))	
										}
										
									}
								}
							}
						}
					} else {
						//TODO: Remove						
						//panic(err)
					}
					cancel()
				} else {
					fmt.Printf("Deferring command to node: %d\n", leader)
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
						fmt.Fprintf(os.Stderr, "SyncPropose returned error %v\n", err)
					}
				} else {
					result, err := nh.SyncRead(ctx, configuration.ClusterID, []byte(key))
					if err != nil {
						fmt.Fprintf(os.Stderr, "SyncRead returned error %v\n", err)
					} else {
						fmt.Fprintf(os.Stdout, "query key: %s, result: %s\n", key, result)
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
