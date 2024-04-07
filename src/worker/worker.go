package main

import (
	"context"
	"fmt"
	"github.com/nireitdev/go-network-distributed-scanner/config"
	db "github.com/nireitdev/go-network-distributed-scanner/db"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	ws          sync.WaitGroup
	Hostname    string
	cfg         *config.Config
	redisclient db.Redisdb
)

type Addr struct {
	ip    string
	port  string
	proto string
}

type Worker struct {
	job  chan Addr
	open chan Addr
}

func main() {

	hostname, err := os.Hostname()
	if err != nil {
		log.Println(err)
	}
	Hostname = hostname

	cfg = config.ReadConfig()

	worker := &Worker{}
	worker.job = make(chan Addr)
	worker.open = make(chan Addr)

	ctx := context.Background()

	//Inicio conexion Redis:
	redisclient = db.Redisdb{Addr: cfg.Redis.Addr,
		User:     cfg.Redis.User,
		Password: cfg.Redis.Pass,
	}
	redisclient.Open(ctx)
	defer redisclient.Close()
	err = redisclient.Publish(db.REPORT_CHANN, "INIT SERVER NRO "+strconv.Itoa(int(redisclient.NroServer))+" host = "+Hostname)
	if err != nil {
		log.Fatalln("Fail publising : ", err)
	}

	err = redisclient.Alive(Hostname)
	if err != nil {
		log.Printf("Error server not alive!")
	}

	go worker.listen()

	for i := 0; i < cfg.Config.NThreads; i++ {
		ws.Add(1)
		go worker.scanner(ctx, i)
	}

	ws.Wait()

	fmt.Printf("fin\n")

}

func (w Worker) scanner(ctx context.Context, n_scanner int) {
	defer ws.Done()

	for addr := range w.job {

		log.Printf("Scanner nro %d Addr %s:%s \n", n_scanner, addr.ip, addr.port)

		hostport := net.JoinHostPort(addr.ip, addr.port)

		if addr.proto == "tcp" {
			conn, err := net.DialTimeout(addr.proto, hostport, 500*time.Millisecond)
			if err != nil {
				//close port, remote server down, firewall, etc
				continue
			}
			defer conn.Close()
		}

		//Fckng UDP motherfckr!
		//RFC1122
		if addr.proto == "udp" {
			s, err := net.ResolveUDPAddr("udp4", hostport)
			conn, err := net.DialUDP("udp4", nil, s)
			if err != nil {
				continue
			}

			defer conn.Close()
			conn.SetDeadline(time.Now().Add(500 * time.Millisecond))

			_, err = conn.Write([]byte("fafafa\n"))
			if err != nil {
				continue
			}
			//espero un byte por udp
			d := make([]byte, 1)
			_, err = conn.Read(d)
			if err != nil {
				continue
			}
		}

		msg := fmt.Sprintf("Open:%s,%s,%s\n", addr.proto, addr.ip, addr.port)
		fmt.Printf(msg)
		err := redisclient.Publish(db.REPORT_CHANN, msg)
		if err != nil {
			log.Fatalf("Error tratando de publicar", err)
		}
	}
}

func (w Worker) listen() {
	//me suscribo al canal y me quedo a la espera de comandos:
	sub := redisclient.Subscribe(db.JOBS_CHANN)
	for msg := range sub {
		var command, params string
		_, err := fmt.Sscanf(msg, "%s %s", &command, &params)
		if err != nil {
			//string no valido
		}

		if command == "ALIVE" {
			redisclient.Alive(Hostname)
		}

		if command == "DONE" {
			for len(w.job) > 0 {
				//
			}
			redisclient.Publish(db.REPORT_CHANN, "DONE "+Hostname)
		}

		if command == Hostname {
			log.Printf("New IP for scan: %s", params)

			//Cargo en el canal de "jobs" los escaneos de ip, puerto y protocolo
			//Luego cada go-routine "scanner()" se encarga de procesarlas concurrentemente
			//Por ahora dejo deshabilitado el escaneo UDP
			//for _, proto := range []string{"tcp" "udp"} {
			for _, proto := range []string{"tcp"} {
				for i := 1; i <= 1024; i++ {
					addr := Addr{params, strconv.Itoa(i), proto}
					w.job <- addr
				}
			}
		}

	}
}
