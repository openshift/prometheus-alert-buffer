package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	storagePath := flag.String("storage-path", "messages.db", "The path for storing message data.")
	listenAddr := flag.String("listen-address", ":9099", "The address to listen on for web requests.")
	retention := flag.Duration("retention", 24*time.Hour, "The retention time after which stored messages will be purged.")
	gcInterval := flag.Duration("gc-interval", 10*time.Minute, "The interval at which to run garbage collection cycles to purge old entries.")
	pushInterval := flag.Duration("push-interval", 5*time.Second, "The interval at which to push messages to websocket clients.")
	flag.Parse()

	log.Fatal(runService(*storagePath, *listenAddr, *retention, *gcInterval, *pushInterval))
}

func runService(storagePath, listenAddr string, retention, gcInterval, pushInterval time.Duration) error {
	registry := prometheus.NewRegistry()
	// Go-specific metrics about the process (GC stats, goroutines, etc.).
	registry.MustRegister(prometheus.NewGoCollector())
	// Go-unrelated process metrics (memory usage, file descriptors, etc.).
	registry.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
	store, err := newBoltStore(&boltStoreOptions{
		path:       storagePath,
		retention:  retention,
		gcInterval: gcInterval,
		registry:   registry,
	})
	if err != nil {
		return fmt.Errorf("Error opening message store:%v", err)
	}
	go store.start()
	defer store.close()

	log.Printf("Listening on %v...", listenAddr)
	return serve(listenAddr, pushInterval, store, registry)
}
