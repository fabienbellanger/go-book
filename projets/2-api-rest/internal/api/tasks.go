package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"example.com/tasksapi/internal/store"
)

// maxBodyBytes borne la taille d'un corps de requête (anti-déni de service).
const maxBodyBytes = 1 << 20 // 1 Mio

// listResponse enveloppe la liste : un objet (plutôt qu'un tableau nu) laisse
// la place d'ajouter des métadonnées (total, page…) sans casser le contrat.
type listResponse struct {
	Tasks []store.Task `json:"tasks"`
	Count int          `json:"count"`
}

// errorBody est le corps JSON renvoyé pour toute erreur.
type errorBody struct {
	Error string `json:"error"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	filter, err := parseListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tasks, err := s.store.List(r.Context(), filter)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse{Tasks: tasks, Count: len(tasks)})
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var in store.TaskInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	t, err := s.store.Create(r.Context(), in)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/tasks/"+strconv.FormatInt(t.ID, 10))
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := s.store.Get(r.Context(), id)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in store.TaskInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := in.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	t, err := s.store.Update(r.Context(), id, in)
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := s.store.Delete(r.Context(), id); err != nil {
		s.storeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseID lit le segment {id} de l'URL (routage Go 1.22) et le valide. En cas
// d'identifiant absent ou non entier positif, il répond 400 et renvoie false.
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "identifiant invalide")
		return 0, false
	}
	return id, true
}

// parseListFilter lit les paramètres de requête ?done, ?limit, ?offset.
func parseListFilter(r *http.Request) (store.ListFilter, error) {
	q := r.URL.Query()
	var f store.ListFilter

	if v := q.Get("done"); v != "" {
		done, err := strconv.ParseBool(v)
		if err != nil {
			return f, errors.New("paramètre done invalide (true/false attendu)")
		}
		f.Done = &done
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, errors.New("paramètre limit invalide")
		}
		f.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, errors.New("paramètre offset invalide")
		}
		f.Offset = n
	}
	return f, nil
}

// storeError traduit une erreur du Store : ErrNotFound => 404, le reste => 500.
func (s *Server) storeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "tâche introuvable")
		return
	}
	s.serverError(w, r, err)
}

// serverError journalise une erreur inattendue et répond 500 sans fuiter de détail.
func (s *Server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.ErrorContext(r.Context(), "erreur interne", "err", err,
		"request_id", requestIDFrom(r.Context()))
	writeError(w, http.StatusInternalServerError, "erreur interne")
}

// writeJSON encode v en JSON avec le code de statut donné.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v) // l'erreur d'écriture réseau n'est pas récupérable ici
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

// decodeJSON lit le corps de la requête dans dst, en bornant sa taille, en
// refusant les champs inconnus, et en exigeant un objet JSON unique.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("le corps contient des données superflues après l'objet JSON")
	}
	return nil
}
