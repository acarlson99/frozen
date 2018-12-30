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
	UName  string
	Nick   string
	Pw     string
	Output chan Message
	CurrentChatRoom ChatRoom
}

type Message struct {
	Username string
	Text     string
}

type ChatServer struct {
	AddUsr     chan User
	AddNick    chan User
	RemoveNick chan User
	NickMap    map[string]User
	Users      map[string]User
	Rooms      map[string]ChatRoom
	Create     chan ChatRoom
	Delete     chan ChatRoom
	UsrJoin    chan Request
	UsrLeave   chan Request
}

type ChatRoom struct {
	Name  string
	Users map[string]User
	Join  chan User
	Leave chan User
	Input chan Message
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
					Name:  request.RoomName,
					Users: make(map[string]User),
					Join:  make(chan User),
					Leave: make(chan User),
					Input: make(chan Message),
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
		Username: sender,
		Text:     msg,
	}
}

func writeToChan(p User, sender, msg string) {
	p.Output <- Message{
		Username: sender,
		Text:     msg,
	}
}

func (room *ChatRoom) Run() {
	for {
		select {
		case user := <-room.Join:
			room.Users[user.UName] = user
			room.Input <- Message{
				Username: "SYSTEM",
				Text:     fmt.Sprintf("%s joined %s", user.Nick, room.Name),
			}
		case user := <-room.Leave:
			delete(room.Users, user.UName)
			room.Input <- Message{
				Username: "System",
				Text:     fmt.Sprintf("%s left %s", user.Nick, room.Name),
			}
		case msg := <-room.Input:
			for _, user := range room.Users {
				select {
				case user.Output <- msg:
				default:
				}
			}
		}
	}
}

func isprint(c byte) bool {
	return (c >= 32 && c <= 126)
}

func San(s string) string {
	s_ := ""
	l := len(s)
	for i := 0; i < l; i++ {
		if isprint(s[i]) {
			s_ += string(s[i])
		}
	}
	return s_
}

func HandleConn(chatServer *ChatServer, conn net.Conn) {
	defer conn.Close()

	io.WriteString(conn, "Enter your Username: ")
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	name := San(scanner.Text())

	var user User
	if tmp, test := chatServer.Users[name]; test {
		user = tmp

		io.WriteString(conn, "Enter your Password: ")
		scanner.Scan()
		pass := San(scanner.Text())
		for pass != user.Pw {
			io.WriteString(conn, "try again:\n")
			scanner.Scan()
			pass = San(scanner.Text())
		}

	} else {
		io.WriteString(conn, "Enter Nickname: ")
		scanner.Scan()
		nickname := San(scanner.Text())

		for {
			if _, test := chatServer.NickMap[nickname]; test {
				io.WriteString(conn, "try again this Nickname is taken\n")
				scanner.Scan()
				nickname = San(scanner.Text())

			} else {
				break
			}

		}

		io.WriteString(conn, "Enter a Password for your account: ")
		scanner.Scan()
		pass := San(scanner.Text())
		tmp := User{
			UName:  name,
			Output: make(chan Message, 10),
			Nick:   nickname,
			Pw:     pass,
		}
		chatServer.AddUsr <- tmp
		user = tmp
	}
	io.WriteString(conn, "Enter Chat Room: ")

	scanner.Scan()

	request := Request{
		Person:   &user,
		RoomName: San(scanner.Text()),
	}
	chatServer.UsrJoin <- request

	defer func() {
		chatServer.UsrLeave <- request
	}()

	go func() {
		for scanner.Scan() {
			ln := San(scanner.Text())
			args := strings.Split(ln, " ")
			if args[0] == "NICK" && len(args) > 1 {
				i := 0
				for _, p := range chatServer.Users {
					if i != 0 {
						break
					} else if p.Nick == args[1] {
						writeToChan(user, "SYSTEM", "nickname \""+args[1]+"\" taken")
						i = 1
					}
				}

				if _, test := chatServer.NickMap[args[1]]; test {
					i = 2
				}
				if i == 0 {
					chatServer.RemoveNick <- user
					delete(chatServer.NickMap, user.Nick)
					chatServer.NickMap[args[1]] = user
					user.Nick = args[1]
					chatServer.RemoveNick <- user
				}
			} else if ln == "WHOAMI" {
				writeToChan(user, "SYSTEM", "\nusername: "+user.UName+"\nnickname: "+user.Nick+"\ncurrent room: "+user.CurrentChatRoom.Name)
			} else if ln == "NAMES" {
				for person := range chatServer.Users {
					writeToChan(user, "SYSTEM", person)
				}
			} else if ln == "ROOMMATES" {
				for _, person := range user.CurrentChatRoom.Users {
					writeToChan(user, "SYSTEM", person.Nick)
				}
			} else if args[0] == "PRIVMSG" && len(args) > 2 {
				if args[1] == "USR" {
					usr, ok := chatServer.Users[args[2]]
					if ok {
						usr.Output <- Message{
							Username: user.UName,
							Text:     fmt.Sprintf("%s", ln),
						}
					} else {
						user.Output <- Message{
							Username: "SYSTEM",
							Text:     fmt.Sprintf("User not found"),
						}
					}
				} else if args[1] == "CHAN" {
					room, ok := chatServer.Rooms[args[2]]
					if ok {
						room.Input <- Message{
							Username: user.UName,
							Text:     ln,
						}
					} else {
						user.Output <- Message{
							Username: user.UName,
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
				for room := range chatServer.Rooms {
					writeToChan(user, "SYSTEM", room)
				}
			} else if args[0] == "JOIN" && len(args) > 1 {
				request = Request{
					Person:   &user,
					RoomName: user.CurrentChatRoom.Name,
				}
				chatServer.UsrLeave <- request
				request = Request{
					Person:   &user,
					RoomName: args[1],
				}
				chatServer.UsrJoin <- request
			} else if ln == "PART" {
				request = Request{
					Person:   &user,
					RoomName: user.CurrentChatRoom.Name,
				}
				chatServer.UsrLeave <- request
				request = Request{
					Person:   &user,
					RoomName: "lobby",
				}
				chatServer.UsrJoin <- request
			} else {
				user.CurrentChatRoom.Input <- Message{user.Nick, ln}
			}
		}
	}()

	for msg := range user.Output {
		if msg.Username != user.UName {
			_, err := io.WriteString(conn, msg.Username+": "+msg.Text+"\n")
			if err != nil {
				break
			}
		}
	}
}

func main() {
	server, err := net.Listen("tcp", ":9000")
	defer server.Close()

	if err != nil {
		log.Fatalln(err.Error())
	}
	chatServer := &ChatServer{
		AddUsr:     make(chan User),
		AddNick:    make(chan User),
		RemoveNick: make(chan User),
		NickMap:    make(map[string]User),
		Users:      make(map[string]User),
		Rooms:      make(map[string]ChatRoom),
		Create:     make(chan ChatRoom),
		Delete:     make(chan ChatRoom),
		UsrJoin:    make(chan Request),
		UsrLeave:   make(chan Request),
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
		go HandleConn(chatServer, conn)
	}
}
