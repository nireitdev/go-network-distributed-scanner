package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/nireitdev/go-network-distributed-scanner/config"
	db "github.com/nireitdev/go-network-distributed-scanner/db"
	"log"
	"net"
	"os"
	"strings"
)

var (
	cfg         *config.Config
	redisclient db.Redisdb
)

func main() {

	if len(os.Args) <= 1 {
		fmt.Printf("%s\n\n", "Error: Invalid input option.")
		os.Exit(1)
	}

	ctx := context.Background()
	cfg = config.ReadConfig()

	redisclient = db.Redisdb{Addr: cfg.Redis.Addr,
		User:     cfg.Redis.User,
		Password: cfg.Redis.Pass,
	}
	redisclient.Open(ctx)
	defer redisclient.Close()

	//Workers alive????
	fmt.Println("Searching for workers....")
	workers := redisclient.GetRemoteWorkers()
	fmt.Printf("Found [ %d ] workers \n", len(workers))

	if len(workers) > 0 {

		for ii := 1; ii < len(os.Args); ii++ {
			//si el argumento es una red con mascara "expando" las IPs:
			if strings.Contains(os.Args[ii], "/") {
				ips := expand(os.Args[ii])
				for j, ip := range ips {
					assignWork(ip, workers, j)
				}
			} else {
				assignWork(os.Args[ii], workers, ii)

			}
		}

		redisclient.Publish(db.JOBS_CHANN, "DONE")

		done := 0
		scan := redisclient.Subscribe(db.REPORT_CHANN)
		for res := range scan {
			if strings.Contains(res, "DONE") {
				done++
				if done == len(workers) {
					break
				}
			} else {
				fmt.Printf(res)
			}
		}
	}

	log.Println("Done.")
}

func assignWork(ipscan string, workers []string, number int) {
	//Simple eleccion del worker utilizando Roud-Robin:
	nro := number % len(workers)
	msg := fmt.Sprintf("%s %s", workers[nro], ipscan)
	err := redisclient.Publish(db.JOBS_CHANN, msg)
	if err != nil {
		log.Println("Fail publising : ", err)
	}
}

// Expande el ip/mask a ips separadas
func expand(cidr string) []string {
	ips := []string{}

	_, ipv4Net, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Println("Invalid option: %", cidr)
		return ips
	}
	// addr+mask => uint32
	mask := binary.BigEndian.Uint32(ipv4Net.Mask)
	start := binary.BigEndian.Uint32(ipv4Net.IP)
	finish := (start & mask) | (mask ^ 0xffffffff)

	for i := start; i <= finish; i++ {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, i)

		ips = append(ips, ip.String())
	}

	return ips
}
