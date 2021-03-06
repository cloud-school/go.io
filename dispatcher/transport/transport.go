package dispatcher_transport

import (
	"go.io/auth/transport"
	"go.io/dispatcher/client"
	"go.io/dispatcher/message"
)

type DispatcherTransport interface {
	Listen(messageChannel chan dispatcher_message.Message, clients *dispatcher_client.Clients)
	Destroy()
}

func NewDispatcherTransport(auth *auth_transport.AuthTransport) DispatcherTransport {
	// TODO: which transport to use should come from env config
	t := NewSockjsDispatcherTransport(auth)
	return DispatcherTransport(&t)
}
