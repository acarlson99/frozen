package main

import (
	"fmt"
	"hash/fnv"
	"net/http"
)

type person struct {
	uname  string
	nick   string
	passwd uint64
}

func hash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func index_handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>OOPSIE WOOPSIE!! Uwu We made a fucky wucky!! A wittle fucko boingo! The code monkeys🙈💻 at our headquarters are working VEWY HAWD to fix this!</h1>")
}

func change_nick(m map[string]person, uname string, s string) {
	for key, value := range m {
		if key != uname && s == value.nick {
			fmt.Printf("nick \"%s\" already taken.  Not changing nick\n", s)
			return
		}
	}
	m[uname] = person{uname: m[uname].uname, nick: s, passwd: m[uname].passwd}
}

func addusr(m map[string]person, p person) {
	for key, value := range m {
		if key == p.uname {
			fmt.Println("username taken:", p.uname)
			return
		} else if value.nick == p.nick {
			fmt.Println("nickname taken:", p.nick)
			return
		}
	}
	m[p.uname] = p
	fmt.Println(p.uname, "successfully added")
}

func main() {
	m := make(map[string]person)
	p1 := person{uname: "a", nick: "bob", passwd: hash("jimbo")}
	p2 := person{uname: "b", nick: "jim", passwd: hash("james")}
	fmt.Println("a")
	addusr(m, p1)
	fmt.Println("b")
	addusr(m, p2)
	fmt.Println("c")
	addusr(m, p1)

	for key, value := range m {
		fmt.Println(key, value)
	}
	fmt.Println("Changing nickname of person b")
	change_nick(m, p2.uname, "bob")

	for key, value := range m {
		fmt.Println(key, value)
	}

	http.HandleFunc("/", index_handler)
	if err := http.ListenAndServe(":8000", nil); err != nil { // NOTE: locally hosted here: 127.0.0.1:8000/
		panic(err)
	}
}
