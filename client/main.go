package main

import (
	pb "MyChat/MyChat"
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"io"
	"log"
	"time"
)

var (
	// Server name
	Server string
	// MyName client name
	MyName = &pb.Name{}
	// Client client
	client pb.ChatServiceClient
)

func init() {
	flag.StringVar(&Server, "server", "127.0.0.1:8080", "grpc server address")
	flag.StringVar(&MyName.Name, "name", "", "name")
	flag.Parse()

	if conn, err := grpc.Dial(Server, grpc.WithInsecure()); err != nil {
		log.Fatal("connect to server failed", err)
	} else {
		client = pb.NewChatServiceClient(conn)
	}
}

var message = make(chan string, 10)

func chat() {
	stream, err := client.Chat(context.Background())
	if err != nil {
		log.Fatalf("%v.Chat() error = _, %v", client, err)
	}
	waitc := make(chan struct{})
	// read goroutine
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive a msg : %v", err)
			}
			log.Printf("======>%s say: %s", in.GetSender(), in.GetMsg())
		}
	}()
	// write goroutine
loop:
	for {
		select {
		case s := <-message:
			if s == "exit" {
				stream.CloseSend()
				return
			}
			stream.Send(&pb.Msg{Sender: MyName.Name, Msg: s})
		case <-waitc:
			break loop
		}
	}
}

// get who are in group
func get() {
	stream, err := client.GetGroup(context.Background(), &pb.GetGroupRequest{})
	if err != nil {
		log.Fatalf("Couldn't get clients in group error :%v", err)
	}
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Printf("Failed to receive a list : %v", err)
			return
		}
		log.Printf("=we=got=%s", in.Name)
	}
}

func input() {
	for {
		var cmd, s string
		fmt.Scan(&cmd, &s)
		//fmt.Println(cmd, s)
		switch cmd {
		case "get":
			get()
		case "msg":
			message <- s
		}
	}
}

func main() {
	// join the group
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	sName, err := client.Join(ctx, MyName)
	if err != nil {
		log.Fatalf("%v.Jion(_) = _, %v", client, err)
	}
	log.Printf("Connected!, server name is: %s\n", sName.Name)

	go input()

	chat()
}
