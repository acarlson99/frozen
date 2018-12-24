package main

import (
	"io"
	"log"
	"net"
	"fmt"
	"bufio"
	"strings"
)

type Request struct {
	Person *User
	RoomName string
}


type User struct {
	Name string
	Nick string
	Pw	 string										// Using this name to go into map (in chatRoom)
	Output chan Message
	CurrentChatRoom ChatRoom 
}

type Message struct {
	Username string									//each message will be given to each user
	Text string
}

type ChatServer struct {
	AddUser chan User
	Users map [string]User
	Room map [string]ChatRoom
	Create chan ChatRoom
	Delete chan ChatRoom
	UserJoin chan Request 
	UserLeave chan Request 
}


type ChatRoom struct {							// think of it as a single chat room ; this struct will handle events
	Name string
	RoomUsers map[string]User							// Map (Users) of string to user (User is value) : make(map[string]string)
	Join chan User									// chan int || chan string || chan struct
	Leave chan User
	Input chan Message
}

func (cs *ChatServer) Run() {
	for {
		select {
			case user := <-cs.AddUser:
				cs.Users[user.Name] = user
			case chatRoom := <-cs.Create:
				cs.Room[chatRoom.Name] = chatRoom
				go chatRoom.Run();				
				go chatRoom.Run();				
				go chatRoom.Run();				
				go chatRoom.Run();				
							
			

			case chatRoom := <-cs.Delete:
				delete(cs.Room, chatRoom.Name)
			case request := <-cs.UserJoin:
				if chatRoom, test := cs.Room[request.RoomName]; test {
					chatRoom.Join <- *(request.Person)	
					request.Person.CurrentChatRoom.Leave <- *(request.Person)
					request.Person.CurrentChatRoom = chatRoom
				} else {
						chatRoome := ChatRoom {
						Name: request.RoomName,
						RoomUsers: make(map[string]User),
						Join: make(chan User),
						Leave: make(chan User),
						Input: make(chan Message),
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

func (cr *ChatRoom) Run() {						// this method will handle all logic for ChatRoom
	for {											// things will connect to it, and come back out of it
		select {									// select lets you wait on multiple channel operations
			case user := <-cr.Join:					// getting from or receiving from
				cr.RoomUsers[user.Name] = user
				cr.Input <- Message {
					Username: "SYSTEM",
					Text: fmt.Sprintf("%s joined", user.Nick),
				}
			case user := <-cr.Leave:
				delete(cr.RoomUsers, user.Name)
				cr.Input <- Message {
					Username: "SYSTEM",
					Text: fmt.Sprintf("%s left", user.Nick),
				}
				
			case msg := <-cr.Input:
				fmt.Printf("Printing")
				for _, user := range cr.RoomUsers {
					fmt.Println(user.Name)
					select {
						case user.Output<- msg:
						default:
					}
				}		
		}
			
	}
}

func writeToChan(p User, sender, msg string) {
	p.Output <- Message {
		Username: sender,
		Text: msg,
	}
}
		
func handleConn(chatServer *ChatServer, conn net.Conn) {				
	defer conn.Close()								
	
	io.WriteString(conn, "Enter your Username: \n")
	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	name := scanner.Text()

	var user User
	if tmp, test := chatServer.Users[name]; test {
		user = tmp
		
		io.WriteString(conn, "Enter your Password:\n")	
		scanner.Scan()
		pass := scanner.Text()
		for (pass != user.Pw) {
			io.WriteString(conn, "try again:\n")	
			scanner.Scan()
			pass = scanner.Text()	
		}
		
		} else {
		io.WriteString(conn, "Enter Nickname: \n")	
		scanner.Scan()
		nickname := scanner.Text()
		io.WriteString(conn, "Enter a PassWord for your account \n")	
		scanner.Scan()
		pass := scanner.Text()
			tmp := User{
			Name: name,
			Output: make(chan Message, 10),
			Nick: nickname,
			Pw: pass,
		}		
		chatServer.AddUser <- tmp
		user = tmp
	}
	io.WriteString(conn, "Enter Chat Room: \n")

	

	scanner.Scan()

	request := Request {
		Person: &user,
		RoomName: scanner.Text(),
	}
	chatServer.UserJoin <- request

	defer func () {
		chatServer.UserLeave <- request
	} ()

	go func() {
		for scanner.Scan() {
			ln := scanner.Text ()
			args := strings.Split(ln, " ")
			if args[0] == "NICK" && len(args) > 1 {
				i := 0
				if len(args[1]) > 32 {
					writeToChan(user, "SYSTEM", "nickname too long\n")
					i = 1
				} else if len(args[1]) < 3 {
					writeToChan(user, "SYSTEM", "nickname too short\n")
					i = 2
				}
				for person := range chatServer.Users {
					if i != 0 {
						break
					} else if person == args[1] {
						writeToChan(user, "SYSTEM", "nickname \"" + args[1] + "\" taken\n")
						i = 3
					}
				}
				if i == 0 {
					user.Nick = args[1]
				}
			} else if args[0] == "WHOAMI" {
				
				writeToChan(user, "SYSTEM", user.Nick + "\n")
			} else if args[0] == "NAMES" {
				for person := range chatServer.Users {
					writeToChan(user, "SYSTEM", person + "\n")
				}
			} else if args[0] == "ROOMMATES" {
				for person := range user.CurrentChatRoom.RoomUsers {
					writeToChan(user, "SYSTEM", person + "\n")
				}
				writeToChan(user, "SYSTEM", "\n")
			} else if args[0] == "PRIVMSG" && len(args) > 3 {
				if args[1] == "USR" {
					usr, ok := chatServer.Users[args[2]]

					if ok {
						usr.Output <- Message {
							Username: user.Name,
							Text: fmt.Sprintf("%s\n", ln),
						}
					} else {
						user.Output <- Message {
							Username: "SYSTEM",
							Text: fmt.Sprintf("User not found\n"),
						}
					}
				} else if args[1] == "CHAN" {
					room, ok := chatServer.Room[args[2]]
					if ok {
						room.Input <- Message {
							Username: user.Name,
							Text: user.Name + " : " + ln,
						}
					} else {
						user.Output <- Message {
							Username: user.Name,
							Text: fmt.Sprintf("Room not found\n"),
						}
					}
				} else {
					user.Output <- Message {
						Username: "SYSTEM",
						Text: fmt.Sprintf("Invalid option\n"),
						}
					}
				} else if args[0] == "LIST" {
					for room := range chatServer.Room {
						writeToChan(user, "SYSTEM", room + "\n")
					}
				} else if args[0] == "JOIN" {
					request = Request {
						Person: &user,
						RoomName: args[1],
					}
					chatServer.UserJoin <- request
				} else if args[0] == "PART" {
					// TODO: leave channel
				} else {
					user.CurrentChatRoom.Input <- Message{user.Name, ln}
				}
			}
		}()

	for msg := range user.Output {
		fmt.Print(msg.Username+": "+msg.Text+"\n")
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
	chatServer := &ChatServer {
		AddUser: make(chan User),
		Users: make(map[string]User),
		Room: make(map[string]ChatRoom),
		Create: make(chan ChatRoom),
		Delete: make(chan ChatRoom),
		UserJoin: make(chan Request), 
		UserLeave: make(chan Request), 
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

				
				
				
				
				
				
				
				
				
				
				
				
				
				
	/*	
	var n string
	io.WriteString(conn, "Enter your Username:")

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	n = scanner.Text()
	user := User{
		Name:   n,
		Nick: "",
		Pw: "",
		Output: make(chan Message, 10),
	}
	io.WriteString(conn, "Enter your nickname:")
	scanner.Scan()
	user.Nick = scanner.Text()
	io.WriteString(conn, "Enter your password:")
	scanner.Scan()
	user.Pw = scanner.Text()
	*/
				
				
				
				
				
				
				
				
				
				
				
				
				
				
				











