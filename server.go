package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

var (
	//for a new user
	enterChannel = make(chan *User)

	//for a user is leaving the chat room
	leavingChannel = make(chan *User)

	//for users sends message,at most 8 messages
	messageChannel = make(chan Message, 8)
)

var (
	globalID      int
	idCounterLock sync.Mutex
)

type User struct {
	ID          int
	Addr        string
	EnterAt     time.Time
	MessageChan chan string //individual user channel
}

//Message is used to ignore itself
type Message struct {
	Owner int
	Msg   string
}

func main() {
	//open a TCP Server
	//open a TCP connection on local machine
	listen, err := net.Listen("tcp", ":2022")
	if err != nil {
		panic(listen)
	}
	log.Println("Server is on....")
	//broadcast message
	//using concurrency
	go broadcast()
	//handling the connection
	//here uses an infinite loop to handle any incoming connection
	for {
		conn, err := listen.Accept() //waiting and accepting connection
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConnection(conn) //passing the connection to the function
	}
}

//broadcast function is used to record chat room message and broadcast the message to users
//there are 3 types of broadcast messages
//1.New user comes in
//2.A user left the chat room
//3.Any message is sent by a user
func broadcast() {
	//need to record user info
	//just a simple storage
	users := make(map[*User]struct{})

	for {
		select {
		case user := <-enterChannel:
			//getting user from enterChannel
			users[user] = struct{}{} //just a simple storage
		case user := <-leavingChannel:
			//need to remove from simple storage
			//and close the user channel,avoid goroutine  leaking problem
			delete(users, user)
			close(user.MessageChan) // closed and sendMessage exited
		case msg := <-messageChannel:
			//send to all user from the map
			for user := range users {
				if user.ID == msg.Owner {
					continue
				}
				user.MessageChan <- msg.Msg
			}
		}
	}

}

func handleConnection(conn net.Conn) {
	//handling incoming connection
	defer conn.Close() //close the connection

	//1.create user information
	newUser := &User{
		ID:          GenerateID(), //using generator to generate a user id
		Addr:        conn.RemoteAddr().String(),
		EnterAt:     time.Now(),
		MessageChan: make(chan string, 8), //create a buffer channel
	}
	//2.create a new goroutine for communication
	go sendMessage(conn, newUser.MessageChan)

	//3.Sending welcome message to the new user
	newUser.MessageChan <- "Welcome to demo chat room"

	//4.Move new user to global recorder
	enterChannel <- newUser //send to broadcast func

	userActive := make(chan struct{}) //we not care about the message that send to the channel

	go func() {
		disConnectionDur := 1 * time.Minute
		//create a timer
		//NOTE : ignore the error
		time := time.NewTimer(disConnectionDur) //end up after 5 minute and sending to its(channel)
		for {
			select {
			case <-time.C:
				conn.Close()
			case <-userActive:
				time.Reset(disConnectionDur)
			}
		}
		//it needs to check whether user active

	}()

	//5.handling user input
	in := bufio.NewScanner(conn)
	for in.Scan() {
		messageChannel <- Message{
			Owner: newUser.ID,
			Msg:   strconv.Itoa(newUser.ID) + ":" + in.Text(),
		} //send and broadcast to all user

		//send active message to channel
		userActive <- struct{}{} //it will reset the timer
	}
	//6.user is leaving the chat room
	leavingChannel <- newUser
	messageChannel <- Message{
		Owner: newUser.ID,
		Msg:   "user:`" + strconv.Itoa(newUser.ID) + "`has left.",
	}
}

//sendMessage ch : read data from channel(read only) ,write is not allowed
//there are 3 types of chan : RW(read and write) RO(read only) WO(write only)
func sendMessage(conn net.Conn, ch <-chan string) {
	//Be careful that this function will run on a new routine,if the channel is not being closed
	//this routine won't exist.
	for msg := range ch { //not break if channel is not closed
		fmt.Fprintln(conn, msg)
	}
}

func GenerateID() int {
	idCounterLock.Lock()
	defer idCounterLock.Unlock()
	globalID++
	return globalID
}
