package main

import "sync"

// resource simule une ressource à libérer (fichier, connexion...). Chaque action
// est journalisée dans un log partagé pour rendre l'ordre observable.
type resource struct {
	name string
	log  *[]string
}

func acquire(name string, log *[]string) resource {
	*log = append(*log, "open:"+name)
	return resource{name, log}
}

func (r resource) use()   { *r.log = append(*r.log, "use:"+r.name) }
func (r resource) Close() { *r.log = append(*r.log, "close:"+r.name) }

// processScoped : BON usage. La closure interne crée une portée par itération,
// donc r.Close() s'exécute à CHAQUE tour, juste après use.
// [a b] -> open:a use:a close:a open:b use:b close:b
func processScoped(names []string) []string {
	var log []string
	for _, n := range names {
		func() {
			r := acquire(n, &log)
			defer r.Close()
			r.use()
		}()
	}
	return log
}

// processDeferInLoop : PIÈGE. Les defer s'empilent et ne se déclenchent qu'au
// retour de la fonction : toutes les ressources restent ouvertes en attendant.
// Le retour est NOMMÉ pour que les Close différés (post-return) soient visibles.
// [a b] -> open:a use:a open:b use:b close:b close:a
func processDeferInLoop(names []string) (log []string) {
	for _, n := range names {
		r := acquire(n, &log)
		defer r.Close() // PIÈGE : tous les Close à la fin, en LIFO
		r.use()
	}
	return // les Close s'exécutent ici, après le corps de la boucle
}

// trace journalise l'entrée MAINTENANT et renvoie une fonction qui journalise la
// sortie. Idiome : `defer trace("f", &log)()` — noter le () final.
func trace(name string, log *[]string) func() {
	*log = append(*log, "enter:"+name)
	return func() { *log = append(*log, "exit:"+name) }
}

// withLock exécute fn en tenant mu ; defer garantit l'unlock même si fn panique.
func withLock(mu *sync.Mutex, fn func()) {
	mu.Lock()
	defer mu.Unlock()
	fn()
}
