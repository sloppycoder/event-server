package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// A single Broker will be created in this program. It is responsible
// for keeping a list of which clients (browsers) are currently attached
// and broadcasting events (messages) to those clients.
//
type Broker struct {
	// Create a map of clients, the keys of the map are the channels
	// over which we can push messages to attached clients.  (The values
	// are just booleans and are meaningless.)
	//
	clients map[chan string]bool

	// Channel into which new clients can be pushed
	//
	newClients chan chan string

	// Channel into which disconnected clients should be pushed
	//
	defunctClients chan chan string

	// Channel into which messages are pushed to be broadcast out
	// to attahed clients.
	//
	messages chan string
}

// This Broker method starts a new goroutine.  It handles
// the addition & removal of clients, as well as the broadcasting
// of messages out to clients that are currently attached.
//
func (b *Broker) Start() {

	go func() {

		for {

			select {

			case s := <-b.newClients:

				// There is a new client attached and we
				// want to start sending them messages.
				b.clients[s] = true
				log.Println("Added new client")

			case s := <-b.defunctClients:

				// A client has dettached and we want to
				// stop sending them messages.
				delete(b.clients, s)
				close(s)

				log.Println("Removed client")

			case msg := <-b.messages:

				// There is a new message to send.  For each
				// attached client, push the new message
				// into the client's message channel.
				for s := range b.clients {
					s <- msg
				}
				log.Println("Broadcast message to %d clients", len(b.clients))
			}
		}
	}()
}

const MaxAccounts = 15

var prevAccounts map[string]*Account

// This Broker method handles and HTTP request at the "/events/" URL.
//
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Make sure that the writer supports flushing.
	//
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Create a new channel, over which the broker can
	// send this client messages.
	messageChan := make(chan string)
	prevAccounts = make(map[string]*Account, MaxAccounts)

	// Add this client to the map of those that should
	// receive updates
	b.newClients <- messageChan

	// Listen to the closing of the http connection via the CloseNotifier
	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		// Remove this client from the map of attached clients
		// when `EventHandler` exits.
		b.defunctClients <- messageChan
		log.Println("HTTP connection just closed.")
	}()

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Don't close the connection, instead loop endlessly.
	for {

		msg, open := <-messageChan

		if !open {
			// If our messageChan was closed, this means that the client has
			// disconnected.
			break
		}

		//fmt.Fprintf(w, "event: message\n")
		fmt.Fprintf(w, "data: %s\n\n", msg)

		f.Flush()
	}

	// Done.
	log.Printf("Finished HTTP request at %s", r.URL.Path)
}

func EventServer() {

	// Make a new Broker instance
	broker := &Broker{
		make(map[chan string]bool),
		make(chan (chan string)),
		make(chan (chan string)),
		make(chan string),
	}

	broker.Start()

	http.Handle("/events/", broker)

	keepalive := 300 //default
	s := os.Getenv("EVENT_SERVER_POLL")
	if s != "" {
		i, err := strconv.Atoi(s)
		if err == nil {
			keepalive = i
		}
	}

	// Query database and identify changed records, then send
	// into the Broker's messages channel and are then broadcast
	// out to any clients that are attached.
	go func() {

		tsLastMessage := time.Now()

		for {
			log.Printf("query top %d accounts for updates", MaxAccounts)

			ctx := context.Background()
			accounts, err := GetTopAccounts(ctx, MaxAccounts)
			if err != nil {
				log.Printf("error from MongoDB %+v", err)
				continue
			}

			if prevAccounts == nil {
				prevAccounts = make(map[string]*Account, MaxAccounts)
			}

			for _, acc := range accounts {
				prev, exists := prevAccounts[acc.AccountId]
				// if account not in the map or new account has different fields
				if !exists || !isSame(prev, acc) {
					b, err := json.Marshal(acc)
					if err == nil {
						prevAccounts[acc.AccountId] = acc
						broker.messages <- string(b)
						tsLastMessage = time.Now()

						log.Printf("Sending updated account %s", acc.AccountId)
						//log.Printf("\nold = %+v\nnew = %+v", prev, acc)
					} else {
						log.Printf("Unable to send changed data to client %+v", err)
					}
				}
			}

			// send an empty object for keep alive every 10s
			if time.Since(tsLastMessage).Seconds() >= 10 {
				broker.messages <- "{}"
				tsLastMessage = time.Now()
				log.Printf("sending empty object for keep-alive")
			}

			time.Sleep(time.Duration(keepalive) * time.Millisecond)
		}
	}()

	http.ListenAndServe(":3102", nil)
}

func isSame(a, b *Account) bool {
	if a.AccountId == b.AccountId &&
		a.Nickname == b.Nickname &&
		a.Currency == b.Currency &&
		a.ProdCode == b.ProdCode &&
		a.ProdName == b.ProdName &&
		a.Servicer == b.Servicer &&
		a.Status == b.Status &&
		a.Balances[0].Amount == b.Balances[0].Amount &&
		a.Balances[0].CreditFlag == b.Balances[0].CreditFlag &&
		a.Balances[1].Amount == b.Balances[1].Amount &&
		a.Balances[1].CreditFlag == b.Balances[1].CreditFlag {
		return true
	}
	return false
}
