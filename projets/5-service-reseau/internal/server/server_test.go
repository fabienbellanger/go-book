package server_test

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"example.com/kvd/internal/client"
	"example.com/kvd/internal/server"
)

// startServer démarre un serveur sur un port libre de la boucle locale et
// renvoie son adresse. Le serveur est arrêté proprement en fin de test.
func startServer(t *testing.T, opts server.Options) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen : %v", err)
	}
	srv := server.New(opts)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx, ln) }()

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Serve a renvoyé une erreur : %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("le serveur ne s'est pas arrêté à temps")
		}
	})
	return ln.Addr().String()
}

func dial(t *testing.T, addr string) *client.Client {
	t.Helper()
	c, err := client.Dial(addr, 2*time.Second)
	if err != nil {
		t.Fatalf("Dial : %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestSetGetDeletePing(t *testing.T) {
	addr := startServer(t, server.Options{})
	c := dial(t, addr)

	if err := c.Ping(); err != nil {
		t.Fatalf("Ping : %v", err)
	}
	if err := c.Set("clé", []byte("valeur")); err != nil {
		t.Fatalf("Set : %v", err)
	}

	v, ok, err := c.Get("clé")
	if err != nil || !ok || string(v) != "valeur" {
		t.Fatalf("Get = (%q, %v, %v), voulu (\"valeur\", true, nil)", v, ok, err)
	}

	if _, ok, _ := c.Get("absente"); ok {
		t.Error("Get sur clé absente doit renvoyer ok=false")
	}

	existed, err := c.Delete("clé")
	if err != nil || !existed {
		t.Fatalf("Delete = (%v, %v), voulu (true, nil)", existed, err)
	}
	if _, ok, _ := c.Get("clé"); ok {
		t.Error("la clé doit avoir disparu après Delete")
	}
}

// TestConcurrentClients lance plusieurs clients en parallèle : chacun écrit puis
// relit sa propre clé. À lancer sous -race pour valider l'absence de course.
func TestConcurrentClients(t *testing.T) {
	addr := startServer(t, server.Options{})

	const clients = 25
	var wg sync.WaitGroup
	errs := make(chan error, clients)
	for i := range clients {
		wg.Go(func() {
			c, err := client.Dial(addr, 2*time.Second)
			if err != nil {
				errs <- err
				return
			}
			defer c.Close()

			key := fmt.Sprintf("clé-%d", i)
			val := fmt.Appendf(nil, "valeur-%d", i)
			if err := c.Set(key, val); err != nil {
				errs <- err
				return
			}
			got, ok, err := c.Get(key)
			if err != nil || !ok || string(got) != string(val) {
				errs <- fmt.Errorf("client %d : got (%q,%v,%v)", i, got, ok, err)
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// TestIdleTimeout vérifie qu'une connexion inactive est fermée par le serveur.
func TestIdleTimeout(t *testing.T) {
	addr := startServer(t, server.Options{IdleTimeout: 100 * time.Millisecond})
	c := dial(t, addr)

	if err := c.Ping(); err != nil {
		t.Fatalf("premier Ping : %v", err)
	}
	time.Sleep(250 * time.Millisecond) // dépasse l'idle timeout
	if err := c.Ping(); err == nil {
		t.Error("la connexion aurait dû être fermée pour inactivité")
	}
}

// TestShutdownRefusesNewConnections vérifie qu'après l'arrêt, plus aucune
// connexion n'est acceptée.
func TestShutdownRefusesNewConnections(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	srv := server.New(server.Options{GraceTimeout: 0})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx, ln) }()

	// Le serveur fonctionne.
	c := dial(t, addr)
	if err := c.Set("a", []byte("1")); err != nil {
		t.Fatalf("Set avant arrêt : %v", err)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Serve : %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("le serveur ne s'est pas arrêté")
	}

	if c2, err := client.Dial(addr, 200*time.Millisecond); err == nil {
		c2.Close()
		t.Error("une connexion après l'arrêt aurait dû être refusée")
	}
}
