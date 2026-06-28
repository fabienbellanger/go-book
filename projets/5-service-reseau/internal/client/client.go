// Package client fournit un client Go pour le service clé-valeur. Chaque appel
// est synchrone (envoi d'une requête, lecture de la réponse) et borné par un
// timeout. Un Client n'est pas sûr pour un usage concurrent : ouvrir une
// connexion par goroutine, ou sérialiser les appels.
package client

import (
	"bufio"
	"errors"
	"net"
	"time"

	"example.com/kvd/internal/protocol"
)

// Client est une connexion à un serveur clé-valeur.
type Client struct {
	conn    net.Conn
	br      *bufio.Reader
	timeout time.Duration
}

// Dial ouvre une connexion vers addr. timeout borne aussi bien la connexion
// que chaque opération ultérieure.
func Dial(addr string, timeout time.Duration) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, br: bufio.NewReader(conn), timeout: timeout}, nil
}

// Close ferme la connexion.
func (c *Client) Close() error { return c.conn.Close() }

// roundTrip envoie une requête et lit la réponse, sous deadlines.
func (c *Client) roundTrip(req protocol.Request) (protocol.Response, error) {
	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	if err := protocol.WriteRequest(c.conn, req); err != nil {
		return protocol.Response{}, err
	}
	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	return protocol.ReadResponse(c.br)
}

// Get lit une clé. ok vaut false si la clé est absente.
func (c *Client) Get(key string) (value []byte, ok bool, err error) {
	resp, err := c.roundTrip(protocol.Request{Op: protocol.OpGet, Key: key})
	if err != nil {
		return nil, false, err
	}
	switch resp.Status {
	case protocol.StatusOK:
		return resp.Value, true, nil
	case protocol.StatusNotFound:
		return nil, false, nil
	default:
		return nil, false, errors.New(resp.Err)
	}
}

// Set écrit une clé.
func (c *Client) Set(key string, value []byte) error {
	resp, err := c.roundTrip(protocol.Request{Op: protocol.OpSet, Key: key, Value: value})
	if err != nil {
		return err
	}
	return statusError(resp)
}

// Delete supprime une clé et indique si elle existait.
func (c *Client) Delete(key string) (existed bool, err error) {
	resp, err := c.roundTrip(protocol.Request{Op: protocol.OpDel, Key: key})
	if err != nil {
		return false, err
	}
	switch resp.Status {
	case protocol.StatusOK:
		return true, nil
	case protocol.StatusNotFound:
		return false, nil
	default:
		return false, errors.New(resp.Err)
	}
}

// Ping vérifie que le serveur répond.
func (c *Client) Ping() error {
	resp, err := c.roundTrip(protocol.Request{Op: protocol.OpPing})
	if err != nil {
		return err
	}
	return statusError(resp)
}

// statusError traduit une réponse non-OK en erreur.
func statusError(resp protocol.Response) error {
	if resp.Status == protocol.StatusError {
		return errors.New(resp.Err)
	}
	return nil
}
