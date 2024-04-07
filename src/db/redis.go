package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
	"time"
)

type Redisdb struct {
	Addr      string
	User      string
	Password  string
	ctx       context.Context
	rds       *redis.Client
	NroServer int64
}

const (
	JOBS_CHANN         = "IPSCAN"
	REPORT_CHANN       = "REPORT"
	KEY_SERVER_NRO     = "SERVERNRO"
	KEY_SET_OF_SERVERS = "SERVERSAVAIL"
)

func (r *Redisdb) Open(ctx context.Context) error {
	r.ctx = ctx

	r.rds = redis.NewClient(&redis.Options{
		Addr:        r.Addr,
		Username:    r.User,
		Password:    r.Password,
		DB:          0, //default db????
		ReadTimeout: -1,
	})
	//Redis responde con un pong:="PONG" y err:=<nil>
	//_ , err := client.Ping(ctx).Result()
	err := r.rds.Ping(r.ctx).Err()
	if err != nil {
		log.Fatal("Redis server not found: ", err)
	}
	//obtengo el ID unico de mi server en Redis "SERVERNRO"
	//utilizando una key con valores incrementables
	//en caso de que no exista SERVERNRO, redis la crea y la setea en 1. Luego, ya existe para los demas srvs
	myid, err := r.rds.Incr(r.ctx, KEY_SERVER_NRO).Result()

	if err != nil {
		log.Fatal("Redis ID key not found: ", err)
	}
	r.NroServer = myid
	return err //@TODO: mejorar la salida del error con wrapping de los msg
}

// Funcion para agregar el worker a la lista de servidores activos
func (r *Redisdb) Alive(hostname string) error {

	//envio data al canal
	err := r.rds.SAdd(r.ctx, KEY_SET_OF_SERVERS, hostname).Err()

	if err != nil {
		log.Fatalf("Redis failed to set worker", err)
	}

	return err
}

func (r *Redisdb) Publish(nameChannelPub string, msg string) error {

	//envio data al canal
	err := r.rds.Publish(r.ctx, nameChannelPub, msg).Err()
	if err != nil {
		log.Fatalf("Redis failed to publish", err)
	}

	return err
}

func (r *Redisdb) Subscribe(nameChannelSub string) chan string {
	//recibo data al canal
	mesg := make(chan string)

	go func() {
		sub := r.rds.Subscribe(r.ctx, nameChannelSub)
		defer sub.Close()
		chsub := sub.Channel()
		for msg := range chsub {
			mesg <- msg.Payload
		}
	}()
	return mesg
}

// Envia un ping a los workers y devuelva la lista de los activos
func (r *Redisdb) GetRemoteWorkers() []string {
	_, err := r.rds.Del(r.ctx, KEY_SET_OF_SERVERS).Result()
	if err != nil {
		log.Fatalln("Error ping-ing workers. Unknown numbers of workers: ", err)
	}

	err = r.Publish(JOBS_CHANN, "ALIVE")
	if err != nil {
		log.Fatalln("Unknown numbers of workers: ", err)
	}

	//tiempo de espera para que respondan los workers
	time.Sleep(5 * time.Second)

	workers, err := r.rds.SMembers(r.ctx, KEY_SET_OF_SERVERS).Result()
	if err != nil {
		//log.fatalF(error!)
		return []string{}
	} else {
		return workers

	}
}

func (r *Redisdb) Close() {
	r.rds.Close()
}
