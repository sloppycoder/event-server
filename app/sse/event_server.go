package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/grpclog"
	"grpc-account-svc/api"
	"grpc-account-svc/app/repo"
	"net/http"
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
				grpclog.Info("Added new client")

			case s := <-b.defunctClients:

				// A client has dettached and we want to
				// stop sending them messages.
				delete(b.clients, s)
				close(s)

				grpclog.Info("Removed client")

			case msg := <-b.messages:

				// There is a new message to send.  For each
				// attached client, push the new message
				// into the client's message channel.
				for s := range b.clients {
					s <- msg
				}
				grpclog.Infof("Broadcast message to %d clients", len(b.clients))
			}
		}
	}()
}

const MaxAccounts = 30

var prevAccounts map[string]*api.Account

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
	prevAccounts = make(map[string]*api.Account, MaxAccounts)

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
		grpclog.Info("HTTP connection just closed.")
	}()

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Don't close the connection, instead loop endlessly.
	for {

		// Read from our messageChan.
		msg, open := <-messageChan

		if !open {
			// If our messageChan was closed, this means that the client has
			// disconnected.
			break
		}

		// Write to the ResponseWriter, `w`.
		fmt.Fprintf(w, "data: Message: %s\n\n", msg)

		// Flush the response.  This is only possible if
		// the repsonse supports streaming.
		f.Flush()
	}

	// Done.
	grpclog.Info("Finished HTTP request at ", r.URL.Path)
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

	// Query database and identify changed records, then send
	// into the Broker's messages channel and are then broadcast
	// out to any clients that are attached.
	go func() {

		tsLastMessage := time.Now()

		for {
			grpclog.Infof("query top %d accounts for updates", MaxAccounts)

			ctx := context.Background()
			accounts, err := repo.GetTopAccounts(ctx, MaxAccounts)
			if err != nil {
				grpclog.Info("error from MongoDB %+v", err)
				continue
			}

			if prevAccounts == nil {
				prevAccounts = make(map[string]*api.Account, MaxAccounts)
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

						grpclog.Infof("Sending updated account %s", acc.AccountId)
						grpclog.Infof("\nold = %+v\nnew = %+v", prev, acc)
					} else {
						grpclog.Infof("Unable to send changed data to client %+v", err)
					}
				}
			}

			// send an empty object for keep alive every 10s
			if time.Since(tsLastMessage).Seconds() >= 10 {
				broker.messages <- "{}"
				tsLastMessage = time.Now()
				grpclog.Infof("sending empty object for keep-alive")
			}

			time.Sleep(300 * time.Millisecond)
		}
	}()

	http.ListenAndServe(":3102", nil)
}

func isSame(a, b *api.Account) bool {
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
