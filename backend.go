package main

import (
	"io"
	"log"
	"net"
)

func handleEcho(port string,conn net.Conn){
	defer conn.Close();

	io.Copy(conn,conn)
}

func startEchoServer(port string) error {

	listener,err:= net.Listen("tcp",":"+port)

	if err != nil{
		log.Fatalf("Server couldn't start on port : %s with error : %v",port,err)
	}

	defer listener.Close();

	for{
		conn,err:= listener.Accept()
		if err != nil{
			log.Printf("Accept error at port %s , with error : %v",port,err)
		}

		go handleEcho(port,conn)
	}

}