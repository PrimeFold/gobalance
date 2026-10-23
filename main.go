package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"context"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main(){
	ports:= []string{"9001","9002","9003"}
	
	

	//Starting the Echo servers by looping through the ports slice on the given ports..
	for _,port:= range ports{
		go func (p string)  {
			_ = startEchoServer(p)
		}(port)
	}

	// Initializing server pool..
	pool:= &ServerPool{}
	for _,p:= range ports{
		u,err:= url.Parse("tcp://localhost:"+p)	
		if err != nil{
			log.Printf("Error parsing port : %s",p)
		}
		pool.AddBackend(
			&Backend{
				URL: u,
				Alive: true,
			},
		)

	}

	

	go func ()  {
		log.Printf("Diagnostic pprof server listening on port : 6060")
		if err:= http.ListenAndServe("localhost:6060",nil); err != nil{
			log.Printf("pprof server error : %v\n",err)
		}
	}()

	proxyL,err:= net.Listen("tcp",":8080")
	if err != nil{
		log.Fatalf("Error starting the proxy server : %v",err)
	}

	defer proxyL.Close()

	ctx,cancel:= context.WithCancel(context.Background())

	defer cancel()

	go pool.StartHealthCheckDaemon(ctx)	

	var activeStreams sync.WaitGroup

	go func ()  {
		for {
			clientConn,err:= proxyL.Accept()
			if err != nil{
				log.Printf("Error accepting the connection : %v",err)
				return
			}
		activeStreams.Add(1)

		go func (conn net.Conn)  {
			defer activeStreams.Done()
			proxyConnection(conn,pool)
		}(clientConn)

		}
	}()

	go func ()  {
		metricsMux:= http.NewServeMux()

		metricsMux.Handle("/metrics",promhttp.Handler())
		log.Println("Prometheus metrics live on http://localhost:9090/metrics")
    	if err := http.ListenAndServe(":9090", metricsMux); err != nil {
    	    log.Printf("Metrics server error: %v\n", err)
    	}
	}()


	//TRAP CTRL+C (Wait for the shutdown signal)
	shutdown:= make(chan os.Signal , 1)
	signal.Notify(shutdown,os.Interrupt,syscall.SIGTERM)

	<-shutdown
	log.Println("\n[SHUTDOWN] Signal received. Commencing graceful drain...")
	
	//Basically the Drain sequence..
	proxyL.Close()
	cancel()
	done:= make(chan struct{})

	go func ()  {
		activeStreams.Wait()
		close(done)
	}()

	select{
	case <- done:
		 log.Println("[SHUTDOWN] All active connections drained cleanly. Exiting with code 0.")
	case <-time.After(15*time.Second):
		log.Println("[SHUTDOWN] 15s timeout expired! Forcing exit.")
	}

	


}


