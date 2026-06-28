// Package protocol définit le protocole binaire maison du service clé-valeur.
//
// Le fil de communication est une suite de *frames* préfixées de leur longueur
// (un entier 32 bits big-endian), suivies du *payload*. Préfixer par la
// longueur résout le problème du « framing » sur TCP : un flux d'octets n'a pas
// de notion de message, c'est à l'application de délimiter les siens.
//
//	+----------------+----------------------+
//	| longueur uint32|       payload        |
//	+----------------+----------------------+
//
// Le payload est une requête ou une réponse, sérialisée par MarshalBinary.
package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// MaxFrameSize borne la taille d'une frame (anti-déni de service : sans limite,
// un client malveillant pourrait annoncer 4 Gio et épuiser la mémoire).
const MaxFrameSize = 16 << 20 // 16 Mio

// errMalformed signale un payload tronqué ou incohérent.
var errMalformed = errors.New("message binaire malformé")

// Op est le code d'une opération demandée par le client.
type Op uint8

const (
	OpGet  Op = iota + 1 // lire une clé
	OpSet                // écrire une clé
	OpDel                // supprimer une clé
	OpPing               // test de vivacité
)

// Status est le code de retour d'une réponse.
type Status uint8

const (
	StatusOK       Status = iota // succès (Value porte le résultat éventuel)
	StatusNotFound               // clé absente
	StatusError                  // erreur (Err porte le message)
)

// Request est une requête client.
type Request struct {
	Op    Op
	Key   string
	Value []byte // utilisé par OpSet
}

// Response est une réponse serveur.
type Response struct {
	Status Status
	Value  []byte // utilisé par StatusOK (résultat d'un GET, "pong"…)
	Err    string // utilisé par StatusError
}

// --- Framing -----------------------------------------------------------------

// WriteFrame écrit un payload préfixé de sa longueur.
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > MaxFrameSize {
		return fmt.Errorf("frame de %d octets dépasse la limite (%d)", len(payload), MaxFrameSize)
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadFrame lit une frame (longueur + payload). io.ReadFull garantit qu'on
// attend bien tous les octets annoncés, même fragmentés en plusieurs segments TCP.
func ReadFrame(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > MaxFrameSize {
		return nil, fmt.Errorf("frame annoncée de %d octets dépasse la limite (%d)", n, MaxFrameSize)
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// --- Sérialisation -----------------------------------------------------------

// MarshalBinary sérialise la requête (sans le préfixe de frame).
func (req Request) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte(byte(req.Op))
	switch req.Op {
	case OpGet, OpDel:
		writeBytes(&b, []byte(req.Key))
	case OpSet:
		writeBytes(&b, []byte(req.Key))
		writeBytes(&b, req.Value)
	case OpPing:
		// pas de charge utile
	default:
		return nil, fmt.Errorf("opération inconnue : %d", req.Op)
	}
	return b.Bytes(), nil
}

// UnmarshalBinary désérialise une requête.
func (req *Request) UnmarshalBinary(data []byte) error {
	r := cursor{data: data}
	op, err := r.readByte()
	if err != nil {
		return err
	}
	req.Op = Op(op)
	switch req.Op {
	case OpGet, OpDel:
		key, err := r.readBytes()
		if err != nil {
			return err
		}
		req.Key = string(key)
	case OpSet:
		key, err := r.readBytes()
		if err != nil {
			return err
		}
		val, err := r.readBytes()
		if err != nil {
			return err
		}
		req.Key = string(key)
		req.Value = bytes.Clone(val) // copie : le buffer source sera réutilisé
	case OpPing:
		// rien
	default:
		return fmt.Errorf("%w : opération %d", errMalformed, op)
	}
	return nil
}

// MarshalBinary sérialise la réponse.
func (resp Response) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte(byte(resp.Status))
	switch resp.Status {
	case StatusOK:
		writeBytes(&b, resp.Value)
	case StatusNotFound:
		// pas de charge utile
	case StatusError:
		writeBytes(&b, []byte(resp.Err))
	default:
		return nil, fmt.Errorf("statut inconnu : %d", resp.Status)
	}
	return b.Bytes(), nil
}

// UnmarshalBinary désérialise une réponse.
func (resp *Response) UnmarshalBinary(data []byte) error {
	r := cursor{data: data}
	st, err := r.readByte()
	if err != nil {
		return err
	}
	resp.Status = Status(st)
	switch resp.Status {
	case StatusOK:
		val, err := r.readBytes()
		if err != nil {
			return err
		}
		resp.Value = bytes.Clone(val)
	case StatusNotFound:
		// rien
	case StatusError:
		msg, err := r.readBytes()
		if err != nil {
			return err
		}
		resp.Err = string(msg)
	default:
		return fmt.Errorf("%w : statut %d", errMalformed, st)
	}
	return nil
}

// --- Messages de haut niveau (framing + sérialisation) -----------------------

// WriteRequest sérialise et encadre une requête.
func WriteRequest(w io.Writer, req Request) error {
	payload, err := req.MarshalBinary()
	if err != nil {
		return err
	}
	return WriteFrame(w, payload)
}

// ReadRequest lit et désérialise une requête.
func ReadRequest(r io.Reader) (Request, error) {
	payload, err := ReadFrame(r)
	if err != nil {
		return Request{}, err
	}
	var req Request
	return req, req.UnmarshalBinary(payload)
}

// WriteResponse sérialise et encadre une réponse.
func WriteResponse(w io.Writer, resp Response) error {
	payload, err := resp.MarshalBinary()
	if err != nil {
		return err
	}
	return WriteFrame(w, payload)
}

// ReadResponse lit et désérialise une réponse.
func ReadResponse(r io.Reader) (Response, error) {
	payload, err := ReadFrame(r)
	if err != nil {
		return Response{}, err
	}
	var resp Response
	return resp, resp.UnmarshalBinary(payload)
}

// --- Aides de (dé)sérialisation ----------------------------------------------

// writeBytes écrit une tranche préfixée de sa longueur (uint32 big-endian).
func writeBytes(b *bytes.Buffer, p []byte) {
	var l [4]byte
	binary.BigEndian.PutUint32(l[:], uint32(len(p)))
	b.Write(l[:])
	b.Write(p)
}

// cursor lit séquentiellement un payload, en vérifiant les bornes à chaque pas.
type cursor struct {
	data []byte
	pos  int
}

func (c *cursor) readByte() (byte, error) {
	if c.pos >= len(c.data) {
		return 0, errMalformed
	}
	b := c.data[c.pos]
	c.pos++
	return b, nil
}

func (c *cursor) readBytes() ([]byte, error) {
	if c.pos+4 > len(c.data) {
		return nil, errMalformed
	}
	n := int(binary.BigEndian.Uint32(c.data[c.pos:]))
	c.pos += 4
	if n < 0 || c.pos+n > len(c.data) {
		return nil, errMalformed
	}
	out := c.data[c.pos : c.pos+n]
	c.pos += n
	return out, nil
}
