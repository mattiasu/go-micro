package main

import (
	"authentication/data"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const webPort = "80"
const rpcPort = "5001"

var counts int

type Config struct {
	Repo   data.Repository
	Client *http.Client
}

func main() {
	log.Printf("Starting API server on port: %s\n", webPort)

	// Connect to the database
	conn := connectToDB()
	if conn == nil {
		log.Panic("Unable to connect to database")
	}

	config := &Config{
		Repo:   data.NewPostgresRepository(conn),
		Client: &http.Client{},
	}

	server := &RPCServer{
		Config: config,
	}

	err := rpc.Register(server)
	if err != nil {
		log.Panic("Error registering RPC server: ", err)
	}
	go config.rpcListen()

	log.Println("Listening on RPC port ", rpcPort)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webPort),
		Handler: config.routes(),
	}

	err = srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Println("Error opening database", err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		log.Println("Error pinging database", err)
		return nil, err
	}

	return db, nil
}

func connectToDB() *sql.DB {
	dns := os.Getenv("DNS")

	for {
		connection, err := openDB(dns)
		if err != nil {
			log.Println("Error connecting to database ...", err)
			counts++
		} else {
			log.Println("Connected to database")
			return connection
		}
		if counts > 10 {
			log.Println(err)
			return nil
		}
		log.Println("Retrying in 5 seconds")
		time.Sleep(5 * time.Second)
		continue
	}
}

func (app *Config) setupRepo(conn *sql.DB) {
	app.Repo = data.NewPostgresRepository(conn)
}

func (app *Config) rpcListen() error {
	log.Println("Starting Auth RPC server on port ", rpcPort)
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", rpcPort))
	if err != nil {
		log.Println("Error starting Auth RPC server: ", err)
		return err
	}
	defer listen.Close()

	for {
		rpcConn, err := listen.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(rpcConn)
	}

}
