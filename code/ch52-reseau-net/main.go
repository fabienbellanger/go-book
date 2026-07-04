// Command ch52-reseau-net illustre le package net : serveur d'écho TCP
// (goroutine par connexion), client avec deadline, cadrage par lignes, UDP en
// datagramme et résolution DNS. Idée directrice : net.Conn est un
// io.ReadWriteCloser (Ch. 41), donc tout l'outillage io/bufio s'y branche sans
// rien connaître du transport.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// serveEcho accepte les connexions sur ln et traite chacune dans sa propre
// goroutine (une goroutine par connexion, Ch. 19 / Ch. 23). La boucle s'arrête
// quand ln est fermé : Accept renvoie alors une erreur non nil.
func serveEcho(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // Listener fermé : fin normale de la boucle d'acceptation
		}
		go handleEcho(conn)
	}
}

// handleEcho lit conn ligne par ligne et renvoie chaque ligne telle quelle.
// conn n'est vu que comme un io.ReadWriteCloser : bufio.Scanner ignore tout du
// transport sous-jacent (TCP, TLS, tube en mémoire...).
func handleEcho(conn net.Conn) {
	defer conn.Close() // TOUJOURS fermer : sinon fuite de descripteur + goroutine
	sc := bufio.NewScanner(conn)
	for sc.Scan() { // ScanLines découpe le FLUX d'octets en lignes
		if _, err := fmt.Fprintln(conn, sc.Text()); err != nil {
			return
		}
	}
}

// echoClientRequest ouvre une connexion TCP vers addr, envoie request (terminé
// par '\n' pour le cadrage) et lit une ligne de réponse. Une deadline ABSOLUE
// protège contre un serveur muet : sans elle, ReadString bloquerait sans fin.
func echoClientRequest(addr, request string) (string, error) {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second)) // instant absolu, pas une durée
	if _, err := fmt.Fprintln(conn, request); err != nil {
		return "", err
	}
	resp, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	return resp[:len(resp)-1], nil // on retire le '\n' de cadrage
}

// dialContext se connecte à addr en respectant l'annulation/timeout de ctx
// (Ch. 22). Dès qu'un context circule, DialContext est préférable à Dial :
// une annulation interrompt la tentative de connexion en cours.
func dialContext(ctx context.Context, addr string) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "tcp", addr)
}

// udpEchoRoundTrip lance un serveur d'écho UDP sur un port éphémère, lui envoie
// payload en un datagramme et renvoie la réponse. UDP est SANS connexion : on
// lit/écrit des datagrammes avec ReadFrom/WriteTo sur un net.PacketConn.
func udpEchoRoundTrip(payload string) (string, error) {
	srv, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer srv.Close()
	go func() {
		buf := make([]byte, 1500) // ~ MTU : un datagramme tient dans un Read
		for {
			n, addr, err := srv.ReadFrom(buf)
			if err != nil {
				return // socket fermée
			}
			srv.WriteTo(buf[:n], addr) // renvoie le datagramme à l'expéditeur
		}
	}()

	client, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer client.Close()
	if _, err := client.WriteTo([]byte(payload), srv.LocalAddr()); err != nil {
		return "", err
	}
	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 1500)
	n, _, err := client.ReadFrom(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

// isTimeout indique si err est un dépassement de deadline réseau. Le motif
// errors.As + net.Error.Timeout() est LA façon idiomatique de distinguer un
// timeout d'une autre erreur d'E/S (Ch. 10).
func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// resolveHost résout un nom via le résolveur par défaut, avec annulation par
// context. NON exercé par les tests (dépendrait d'un DNS externe) : illustre
// seulement l'API net.Resolver.
func resolveHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

func main() {
	// Serveur d'écho TCP sur un port éphémère (127.0.0.1:0 => l'OS choisit).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Println("listen:", err)
		return
	}
	defer ln.Close()
	go serveEcho(ln)

	addr := ln.Addr().String() // adresse réellement attribuée, port compris
	reply, err := echoClientRequest(addr, "bonjour")
	fmt.Printf("TCP écho %q -> %q (err=%v)\n", "bonjour", reply, err)

	// UDP en loopback.
	udpReply, err := udpEchoRoundTrip("ping")
	fmt.Printf("UDP écho %q -> %q (err=%v)\n", "ping", udpReply, err)

	// Manipulation d'adresses (purement local, sans I/O).
	host, port, _ := net.SplitHostPort(addr)
	fmt.Printf("SplitHostPort(%q) -> host=%q port=%q\n", addr, host, port)
	fmt.Printf("IP loopback ? %v\n", net.ParseIP(host).IsLoopback())

	// Résolution DNS (peut échouer hors ligne : on n'échoue pas le programme).
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if ips, err := resolveHost(ctx, "localhost"); err == nil {
		fmt.Printf("localhost -> %v\n", ips)
	}
}
