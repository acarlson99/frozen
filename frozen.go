package main

import (
	"bufio"
	"fmt"
//	"io"
	"log"
	"net"
//	"strings"
)

type Request struct {
	Person		*User
	RoomName	string
}

type User struct {
	UName			string
	Nick			string
	Pw				string
	Output			chan Message
//	URooms			map[string]ChatRoom
	CurrentChatRoom ChatRoom
}

type Message struct {
	Username string //each message will be given to each user
	Text     string
}

type ChatServer struct {
	AddUsr		chan User
	AddNick		chan User
	RemoveNick	chan User
	NickMap		map[string]User
	Users		map[string]User
	Rooms		map[string]ChatRoom
	Create		chan ChatRoom
	Delete		chan ChatRoom
	UsrJoin	chan Request
	UsrLeave	chan Request
}

type ChatRoom struct {
	Name		string
	Users		map[string]User
	Join		chan User
	Leave		chan User
	Input		chan Message
}

func (cs *ChatServer) Run() {
	for {
		select {
		case user := <-cs.RemoveNick:
			delete(cs.NickMap, user.Nick)
		case user := <-cs.AddNick:
						cs.NickMap[user.Nick] = user
		case user := <-cs.AddUsr:
			cs.Users[user.UName] = user
			cs.NickMap[user.Nick] = user
		case chatRoom := <-cs.Create:
			cs.Rooms[chatRoom.Name] = chatRoom
			go chatRoom.Run()
			go chatRoom.Run()
			go chatRoom.Run()
			go chatRoom.Run()
		case chatRoom := <-cs.Delete:
			delete(cs.Rooms, chatRoom.Name)
		case request := <-cs.UsrJoin:
			if chatRoom, test := cs.Rooms[request.RoomName]; test {
				chatRoom.Join <- *(request.Person)
				request.Person.CurrentChatRoom = chatRoom
			} else {
				chatRoome := ChatRoom{
					Name:      request.RoomName,
					Users: make(map[string]User),
					Join:      make(chan User),
					Leave:     make(chan User),
					Input:     make(chan Message),
				}
				cs.Rooms[chatRoome.Name] = chatRoome
				cs.Create <- chatRoome
				chatRoome.Join <- *(request.Person)
				request.Person.CurrentChatRoom = chatRoome
			}
		case request := <-cs.UsrLeave:
			room := cs.Rooms[request.RoomName]
			room.Leave <- *(request.Person)
		}
	}
}

func UsrWrite(p User, sender, msg string) {
	p.Output <- Message{
		Username:	sender,
		Text:		msg,
	}
}

func (room *ChatRoom) Run() {
	for {
		select {
		case user := <-room.Join:
			room.Users[user.UName] = user
			room.Input <- Message{
				Username:	"SYSTEM",
				Text:		fmt.Sprintf("%s joined %s", user.Nick, room.Name),
			}
		case user := <-room.Leave:
			delete(room.Users, user.UName)
			room.Input <- Message{
				Username:	"System",
				Text:		fmt.Sprintf("%s left %s", user.Nick, room.Name),
			}
		case msg:= <-room.Input:
			fmt.Printf("Printing")	// TODO: What does this do?
			for _, user := range room.Users {
				fmt.Println(user.Nick)	// TODO: ?
				select {
				case user.Output <- msg:	// TODO: ?
				default:
				}
			}
		}
	}
}

func	HandleConn(chatServer *ChatServer, conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for {
		scanner.Scan()
		fmt.Printf("%s", scanner.Text())
	}
}

func	main() {
	server, err := net.Listen("tcp", ":9000")
	defer server.Close()

	if err != nil {
		log.Fatalln(err.Error())
	}
	chatServer := &ChatServer{
		AddUsr:    make(chan User),
		AddNick:    make(chan User),
		RemoveNick: make(chan User),
		NickMap:	make(map[string]User),
		Users:      make(map[string]User),
		Rooms:		make(map[string]ChatRoom),
		Create:     make(chan ChatRoom),
		Delete:     make(chan ChatRoom),
		UsrJoin:   make(chan Request),
		UsrLeave:  make(chan Request),
	}
	go chatServer.Run()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatalln(err.Error())
		}
		go HandleConn(chatServer, conn)
	}
}
