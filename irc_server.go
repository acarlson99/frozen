package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type Request struct {
	Person   *User
	RoomName string
}

type User struct {
	Name            string
	Nick            string
	Pw              string // Using this name to go into map (in chatRoom)
	Output          chan Message
	CurrentChatRoom ChatRoom
}

type Message struct {
	Username string //each message will be given to each user
	Text     string
}

type ChatServer struct {
	AddUser    chan User
	AddNick    chan User
	RemoveNick chan User
	NickMap    map[string]User
	Users      map[string]User
	Room       map[string]ChatRoom
	Create     chan ChatRoom
	Delete     chan ChatRoom
	UserJoin   chan Request
	UserLeave  chan Request
}

type ChatRoom struct { // think of it as a single chat room ; this struct will handle events
	Name      string
	RoomUsers map[string]User // Map (Users) of string to user (User is value) : make(map[string]string)
	Join      chan User       // chan int || chan string || chan struct
	Leave     chan User
	Input     chan Message
}

func (cs *ChatServer) Run() {
	for {
		select {
		case user := <-cs.RemoveNick:
			delete(cs.NickMap, user.Nick)
		case user := <-cs.AddNick:
			cs.NickMap[user.Nick] = user
		case user := <-cs.AddUser:
			cs.Users[user.Name] = user
			cs.NickMap[user.Nick] = user
		case chatRoom := <-cs.Create:
			cs.Room[chatRoom.Name] = chatRoom
			go chatRoom.Run()
			go chatRoom.Run()
			go chatRoom.Run()
			go chatRoom.Run()

		case chatRoom := <-cs.Delete:
			delete(cs.Room, chatRoom.Name)
		case request := <-cs.UserJoin:
			if chatRoom, test := cs.Room[request.RoomName]; test {
				chatRoom.Join <- *(request.Person)
				request.Person.CurrentChatRoom = chatRoom
			} else {
				chatRoome := ChatRoom{
					Name:      request.RoomName,
					RoomUsers: make(map[string]User),
					Join:      make(chan User),
					Leave:     make(chan User),
					Input:     make(chan Message),
				}
				cs.Room[chatRoome.Name] = chatRoome
				cs.Create <- chatRoome
				chatRoome.Join <- *(request.Person)
				request.Person.CurrentChatRoom = chatRoome
			}
		case request := <-cs.UserLeave:
			room := cs.Room[request.RoomName]
			room.Leave <- *(request.Person)

		}
	}
}

func (cr *ChatRoom) Run() { // this method will handle all logic for ChatRoom
	for { // things will connect to it, and come back out of it
		select { // select lets you wait on multiple channel operations
		case user := <-cr.Join: // getting from or receiving from
			cr.RoomUsers[user.Name] = user
			cr.Input <- Message{
				Username: "SYSTEM",
				Text:     fmt.Sprintf("%s joined %s", user.Nick, cr.Name),
			}
		case user := <-cr.Leave:
			delete(cr.RoomUsers, user.Name)
			cr.Input <- Message{
				Username: "SYSTEM",
				Text:     fmt.Sprintf("%s left", user.Nick),
			}

		case msg := <-cr.Input:
			fmt.Printf("Printing")
			for _, user := range cr.RoomUsers {
				fmt.Println(user.Nick)
				select {
				case user.Output <- msg:
				default:
				}
			}
		}

	}
}

func writeToChan(p User, sender, msg string) {
	p.Output <- Message{
		Username: sender,
		Text:     msg,
	}
}

func handleConn(chatServer *ChatServer, conn net.Conn) {
	defer conn.Close()

	io.WriteString(conn, "Enter your Username: ")
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	name := scanner.Text()

	var user User
	if tmp, test := chatServer.Users[name]; test {
		user = tmp

		io.WriteString(conn, "Enter your Password: ")
		scanner.Scan()
		pass := scanner.Text()
		for pass != user.Pw {
			io.WriteString(conn, "try again:\n")
			scanner.Scan()
			pass = scanner.Text()
		}

	} else {
		io.WriteString(conn, "Enter Nickname: ")
		scanner.Scan()
		nickname := scanner.Text()

		for {
			if _, test := chatServer.NickMap[nickname]; test {
				io.WriteString(conn, "try again this Nickname is taken\n")
				scanner.Scan()
				nickname = scanner.Text()

			} else {
				break
			}

		}

		io.WriteString(conn, "Enter a Password for your account ")
		scanner.Scan()
		pass := scanner.Text()
		tmp := User{
			Name:   name,
			Output: make(chan Message, 10),
			Nick:   nickname,
			Pw:     pass,
		}
		chatServer.AddUser <- tmp
		user = tmp
	}
	io.WriteString(conn, "Enter Chat Room: \n")

	scanner.Scan()

	request := Request{
		Person:   &user,
		RoomName: scanner.Text(),
	}
	chatServer.UserJoin <- request

	defer func() {
		chatServer.UserLeave <- request
	}()

	go func() {
		for scanner.Scan() {
			ln := scanner.Text()
			args := strings.Split(ln, " ")
			if args[0] == "NICK" && len(args) > 1 {
				i := 0
				if len(args[1]) > 32 {
					writeToChan(user, "SYSTEM", "nickname too long")
					i = 1
				} else if len(args[1]) < 3 {
					writeToChan(user, "SYSTEM", "nickname too short")
					i = 2
				}
				for _, p := range chatServer.Users {
					if i != 0 {
						break
					} else if p.Nick == args[1] {
						writeToChan(user, "SYSTEM", "nickname \""+args[1]+"\" taken")
						i = 3
					}
				}

				if _, test := chatServer.NickMap[args[1]]; test {
					i = 3
				}
				if i == 0 {
					chatServer.RemoveNick <- user
					// delete(chatServer.NickMap, user.Nick)
					chatServer.NickMap[args[1]] = user
					user.Nick = args[1]
					chatServer.RemoveNick <- user
				}
			} else if ln == "WHOAMI" {
				writeToChan(user, "SYSTEM", "\nusername: "+user.Name+"\nnickname: "+user.Nick+"\ncurrent room: "+user.CurrentChatRoom.Name)
			} else if ln == "NAMES" {
				for person := range chatServer.Users {
					writeToChan(user, "SYSTEM", person)
				}
			} else if ln == "ROOMMATES" {
				for _, person := range user.CurrentChatRoom.RoomUsers {
					writeToChan(user, "SYSTEM", person.Nick)
				}
			} else if args[0] == "PRIVMSG" && len(args) > 2 {
				if args[1] == "USR" {
					usr, ok := chatServer.Users[args[2]]
					if ok {
						usr.Output <- Message{
							Username: user.Name,
							Text:     fmt.Sprintf("%s", ln),
						}
					} else {
						user.Output <- Message{
							Username: "SYSTEM",
							Text:     fmt.Sprintf("User not found"),
						}
					}
				} else if args[1] == "CHAN" {
					room, ok := chatServer.Room[args[2]]
					if ok {
						room.Input <- Message{
							Username: user.Name,
							Text:     ln,
						}
					} else {
						user.Output <- Message{
							Username: user.Name,
							Text:     fmt.Sprintf("Room not found"),
						}
					}
				} else {
					user.Output <- Message{
						Username: "SYSTEM",
						Text:     fmt.Sprintf("Invalid option"),
					}
				}
			} else if ln == "LIST" {
				for room := range chatServer.Room {
					writeToChan(user, "SYSTEM", room)
				}
			} else if args[0] == "JOIN" && len(args) > 1 {
				request = Request{
					Person:   &user,
					RoomName: user.CurrentChatRoom.Name,
				}
				chatServer.UserLeave <- request
				request = Request{
					Person:   &user,
					RoomName: args[1],
				}
				chatServer.UserJoin <- request
			} else if ln == "PART" {
				request = Request{
					Person:   &user,
					RoomName: user.CurrentChatRoom.Name,
				}
				chatServer.UserLeave <- request
				request = Request{
					Person:   &user,
					RoomName: "lobby",
				}
				chatServer.UserJoin <- request
			} else {
				user.CurrentChatRoom.Input <- Message{user.Nick, ln}
			}
		}
	}()

	for msg := range user.Output {
		fmt.Print(msg.Username + ": " + msg.Text + "\n")
		if msg.Username != user.Name {
			_, err := io.WriteString(conn, msg.Username+": "+msg.Text+"\n")
			if err != nil {
				break
			}
		}
	}
}

func main() {
	server, err := net.Listen("tcp", ":9000")

	if err != nil {
		log.Fatalln(err.Error())
	}
	defer server.Close()
	chatServer := &ChatServer{
		AddNick:    make(chan User),
		RemoveNick: make(chan User),
		AddUser:    make(chan User),
		NickMap:    make(map[string]User),
		Users:      make(map[string]User),
		Room:       make(map[string]ChatRoom),
		Create:     make(chan ChatRoom),
		Delete:     make(chan ChatRoom),
		UserJoin:   make(chan Request),
		UserLeave:  make(chan Request),
	}
	go chatServer.Run()
	go chatServer.Run()
	go chatServer.Run()
	go chatServer.Run()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatalln(err.Error())
		}
		go handleConn(chatServer, conn)
	}
}
