package main

import (
	pb "MyChat/MyChat"
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type chatService struct{}

var (
	// Listen port
	Listen = ":8080"
	//MyName server name
	MyName = &pb.Name{}
	// Group Client list
	Group = []*pb.Name{}
	// GroupLock locker
	GroupLock = sync.RWMutex{}
)

// Clients record how many clients are chatting
type Clients struct {
	sync.RWMutex
	Msg map[pb.ChatService_ChatServer]chan *pb.Msg
}

var clients = &Clients{Msg: map[pb.ChatService_ChatServer]chan *pb.Msg{}}

func (c *Clients) add(scs pb.ChatService_ChatServer, sendMsg chan *pb.Msg) {
	clients.Lock()
	clients.Msg[scs] = sendMsg
	clients.Unlock()
}

func (c *Clients) del(scs pb.ChatService_ChatServer) {
	clients.Lock()
	delete(clients.Msg, scs)
	clients.Unlock()
}

func (cs *chatService) Join(context context.Context, name *pb.Name) (*pb.Name, error) {
	Group = append(Group, name)
	log.Printf("%s has joined in the group !\n", name.Name)
	return MyName, nil
}
func (cs *chatService) GetGroup(ggr *pb.GetGroupRequest, ggs pb.ChatService_GetGroupServer) error {
	GroupLock.RLock()
	defer GroupLock.RUnlock()
	for _, name := range Group {
		if err := ggs.Send(name); err != nil {
			return err
		}
	}
	return nil
}

func (cs *chatService) Chat(cscs pb.ChatService_ChatServer) error {
	message := make(chan *pb.Msg, 10)
	clients.add(cscs, message)
	defer func() {
		clients.del(cscs)
	}()
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := cscs.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Printf("receive error : %v\n", err)
				close(waitc)
				return
			}
			log.Printf("======>%s say: %s\n", in.GetSender(), in.GetMsg())
			broadCast(in)
		}
	}()
loop:
	for {
		select {
		case s := <-message:
			cscs.Send(s)
		case <-waitc:
			break loop
		}
	}
	return nil
}

// get message and broadcast
func readMsg() {
	for {
		var s string
		fmt.Scanln(&s)
		if strings.Trim(s, "\n\r ") != "" {
			broadCast(&pb.Msg{Sender: MyName.Name, Msg: s})
		}
	}
}

// broadcast message to all
func broadCast(s *pb.Msg) {
	clients.RLock()
	for _, v := range clients.Msg {
		select {
		case v <- s:
		default:
			log.Println("Channel is full")
		}
	}
	clients.RUnlock()
}

func start() {
	lis, err := net.Listen("tcp", Listen)
	if err != nil {
		log.Fatalf("Listen error: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterChatServiceServer(s, &chatService{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Serve error: %v", err)
	}
}

func init() {
	flag.StringVar(&MyName.Name, "name", "", "MyName")
	flag.Parse()
}

func main() {
	go readMsg()
	go start()
	select {}
}
