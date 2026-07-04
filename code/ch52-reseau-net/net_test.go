package main

import (
	"context"
	"net"
	"testing"
	"time"
)

// startTestServer lance le serveur d'écho sur un port éphémère du loopback et
// renvoie son adresse. Le Listener est fermé automatiquement en fin de test,
// ce qui termine serveEcho.
func startTestServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen : %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go serveEcho(ln)
	return ln.Addr().String()
}

// TestEchoRoundTrip : le serveur renvoie chaque ligne à l'identique.
func TestEchoRoundTrip(t *testing.T) {
	addr := startTestServer(t)
	got, err := echoClientRequest(addr, "bonjour")
	if err != nil {
		t.Fatalf("requête : %v", err)
	}
	if got != "bonjour" {
		t.Errorf("réponse = %q, veut %q", got, "bonjour")
	}
}

// TestDialContextCanceled : un context déjà annulé fait échouer DialContext
// sans attendre le timeout de connexion.
func TestDialContextCanceled(t *testing.T) {
	addr := startTestServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annulé AVANT le dial
	if _, err := dialContext(ctx, addr); err == nil {
		t.Fatal("DialContext aurait dû échouer sur un context annulé")
	}
}

// TestReadDeadlineExpires : sans données envoyées par le pair, une lecture avec
// deadline courte expire, et isTimeout le reconnaît.
func TestReadDeadlineExpires(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen : %v", err)
	}
	defer ln.Close()
	// Serveur silencieux : il accepte mais n'envoie jamais rien.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		time.Sleep(500 * time.Millisecond) // garde la connexion ouverte, muette
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial : %v", err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	_, err = conn.Read(make([]byte, 16))
	if !isTimeout(err) {
		t.Fatalf("erreur = %v, voulait un timeout réseau", err)
	}
}

// TestUDPEchoRoundTrip : aller-retour d'un datagramme en loopback.
func TestUDPEchoRoundTrip(t *testing.T) {
	got, err := udpEchoRoundTrip("ping")
	if err != nil {
		t.Fatalf("UDP : %v", err)
	}
	if got != "ping" {
		t.Errorf("réponse = %q, veut %q", got, "ping")
	}
}
