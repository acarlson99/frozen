package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

var (
	people = make(People)
	numGen = count(1)
)

type Person struct {
	id	   int
	uname  string
	nick   string
	passwd string
	online bool
	output chan<- string
}

type People map[string] *Person

func (p People) changeNick(uname string, newNick string) error {
	if len(newNick) > 32 {
		return fmt.Errorf("your nickname must be shorter than 32 chars")
	}
	if len(newNick) < 3 {
		return fmt.Errorf("your nickname must be longer than 3 chars")
	}
	for person := range p {
		if p[person].nick == newNick {
			return fmt.Errorf("%s is already used", newNick)
		}
	}
	if person, ok := p[uname]; ok {
		person.nick = newNick
	} else {
		panic("user not found")
	}
	return nil
}


func indexHandler(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "<h1>OOPSIE WOOPSIE!! Uwu We made a fucky wucky!! A wittle fucko boingo! The code monkeys🙈💻 at our headquarters are working VEWY HAWD to fix this!</h1>")
	if err != nil {
		panic(err)
	}
}

func main() {
	server, err := net.Listen("tcp4", ":6667")
	if err != nil {
		panic(err)
	}
	defer server.Close()
	for {
		c, err := server.Accept()
		if err != nil {
			println(err.Error())
		}
		 go handleConnection(c)
	}
}

func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	uname := ""
	tempnick := ""
	// todo create user var that is temporary at first, and then points to the actual user in people map
	for {
		bytes, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				if uname != "" {
					people[uname].online = false
				}
				return
			}
			println(err)
		}
		str := string(bytes)
		args := strings.Split(str, " ")
		if args[0] == "NICK" {
			if uname == "" {
				tempnick = args[1]
				broadcast(fmt.Sprintf(":unnamed_user!%s NICK :%s\n", conn.RemoteAddr(), tempnick))
			} else {
				if err := people.changeNick(uname, args[1]); err != nil {
					conn.Write([]byte(err.Error()))
				}
			}
		} else if args[0] == "USER" {
			uname = args[1]
			user, ok := people[uname]
			if !ok {
				user = &Person{<-numGen, args[1], tempnick, "abc", true, createWriterChannel(conn)}
				people[args[1]] = user
			}
			user.output <- ":irc.42.us.org NOTICE * :Registered\n"
			user.output <- fmt.Sprintf(":irc.42.us.org %03d %s :Your ip is %s\n", user.id, user.nick, conn.RemoteAddr())
		} else if args[0] == "ISON" {
			for i := 1; i < len(args); i++ {
				// TODO handle multiple
				person, ok := people[args[i]]
				if !ok {
					continue
				}
				if person.online {
					conn.Write([]byte(person.uname)) // todo use user var
					break
				}
			}
			conn.Write([]byte("\n")) // todo same
		} else if args[0] == "PRIVMSG" {
			person := people[args[1]]
			if person == nil || !person.online {
				// TODO person not online
			} else {
				person.output <- "PRIVMSG " + uname + " " + strings.Join(args[2:], " ") // TODO this is the wrong message
			}
		} else {
			println(str)
		}
	}
}

func count(start int) chan int {
	ch := make(chan int)

	go func () {
		for i := start;; i++ {
			ch <- i
		}
	}()

	return ch
}

func broadcast(s string) {
	// TODO handle non-registered users (not in the map)
	for uname := range people {
		user := people[uname]
		if user.online {
			user.output <- s
		}
	}
}

func createWriterChannel(conn net.Conn) chan<- string {
	c := make(chan string)
	go func() {
		_, _ = conn.Write([]byte(<-c + "\n"))
	}()
	return c
}