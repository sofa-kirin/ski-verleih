// Package api stellt den Store als JSON-REST-API bereit.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"skiverleih/internal/domain"
	"skiverleih/internal/store"
)

const maxAusweisBytes = 5 << 20

var erlaubteAusweisTypen = map[string]bool{"image/jpeg": true, "image/png": true, "application/pdf": true}

// Server verbindet HTTP-Routen mit dem Store.
type Server struct {
	store *store.Store
	now   func() time.Time
}

// New erzeugt einen Server; now liefert die aktuelle Zeit (in Tests fest).
func New(s *store.Store, now func() time.Time) *Server {
	return &Server{store: s, now: now}
}

// Routes registriert alle API-Routen.
func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/dashboard", s.dashboard)

	mux.HandleFunc("GET /api/ski", s.listeSki)
	mux.HandleFunc("POST /api/ski", s.speichereSki)
	mux.HandleFunc("PUT /api/ski/{id}", s.speichereSki)
	mux.HandleFunc("DELETE /api/ski/{id}", s.loescheSki)
	mux.HandleFunc("GET /api/ski/verfuegbar", s.verfuegbar)

	mux.HandleFunc("GET /api/personen", s.listePersonen)
	mux.HandleFunc("POST /api/personen", s.speicherePerson)
	mux.HandleFunc("PUT /api/personen/{id}", s.speicherePerson)
	mux.HandleFunc("DELETE /api/personen/{id}", s.loeschePerson)
	mux.HandleFunc("POST /api/personen/{id}/ausweis", s.ladeAusweisHoch)
	mux.HandleFunc("GET /api/personen/{id}/ausweis", s.zeigeAusweis)

	mux.HandleFunc("GET /api/ausleihen", s.listeAusleihen)
	mux.HandleFunc("POST /api/ausleihen", s.neueAusleihe)
	mux.HandleFunc("POST /api/ausleihen/{id}/rueckgabe", s.rueckgabe)
	mux.HandleFunc("POST /api/ausleihen/{id}/storno", s.storno)

	mux.HandleFunc("GET /api/preise", s.preise)
	mux.HandleFunc("PUT /api/preise", s.setzePreise)
	mux.HandleFunc("GET /api/preise/berechnen", s.berechnePreis)
	mux.HandleFunc("GET /api/ausweistypen", func(w http.ResponseWriter, r *http.Request) {
		schreibeJSON(w, http.StatusOK, domain.Ausweistypen)
	})
}

// ---------- Hilfsfunktionen ----------

func schreibeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func schreibeFehler(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, store.ErrNichtGefunden):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrKonflikt):
		status = http.StatusConflict
	case errors.Is(err, store.ErrUngueltig):
		status = http.StatusBadRequest
	}
	schreibeJSON(w, status, map[string]string{"fehler": err.Error()})
}

func ungueltig(format string, args ...any) error {
	return fmt.Errorf("%w: %s", store.ErrUngueltig, fmt.Sprintf(format, args...))
}

func liesJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return ungueltig("JSON konnte nicht gelesen werden: %v", err)
	}
	return nil
}

func pfadID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		return 0, ungueltig("ungültige ID %q", r.PathValue("id"))
	}
	return id, nil
}

func zeitraum(r *http.Request) (time.Time, time.Time, error) {
	von, err := time.Parse(time.RFC3339, r.URL.Query().Get("von"))
	if err != nil {
		return time.Time{}, time.Time{}, ungueltig("Parameter 'von' fehlt oder ist kein RFC-3339-Zeitpunkt")
	}
	bis, err := time.Parse(time.RFC3339, r.URL.Query().Get("bis"))
	if err != nil {
		return time.Time{}, time.Time{}, ungueltig("Parameter 'bis' fehlt oder ist kein RFC-3339-Zeitpunkt")
	}
	return von, bis, nil
}

// ---------- Handler ----------

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	schreibeJSON(w, http.StatusOK, s.store.Dashboard(s.now()))
}

func (s *Server) listeSki(w http.ResponseWriter, r *http.Request) {
	schreibeJSON(w, http.StatusOK, s.store.Ski())
}

func (s *Server) speichereSki(w http.ResponseWriter, r *http.Request) {
	var ski domain.Ski
	if err := liesJSON(r, &ski); err != nil {
		schreibeFehler(w, err)
		return
	}
	status := http.StatusCreated
	ski.ID = 0
	if r.Method == http.MethodPut {
		id, err := pfadID(r)
		if err != nil {
			schreibeFehler(w, err)
			return
		}
		ski.ID, status = id, http.StatusOK
	}
	res, err := s.store.SpeichereSki(ski)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, status, res)
}

func (s *Server) loescheSki(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err == nil {
		err = s.store.LoescheSki(id)
	}
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) verfuegbar(w http.ResponseWriter, r *http.Request) {
	von, bis, err := zeitraum(r)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	res, err := s.store.Verfuegbar(von, bis, s.now())
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	if res == nil {
		res = []store.Angebot{}
	}
	schreibeJSON(w, http.StatusOK, res)
}

func (s *Server) listePersonen(w http.ResponseWriter, r *http.Request) {
	schreibeJSON(w, http.StatusOK, s.store.Personen())
}

func (s *Server) speicherePerson(w http.ResponseWriter, r *http.Request) {
	var p domain.Person
	if err := liesJSON(r, &p); err != nil {
		schreibeFehler(w, err)
		return
	}
	status := http.StatusCreated
	p.ID = 0
	if r.Method == http.MethodPut {
		id, err := pfadID(r)
		if err != nil {
			schreibeFehler(w, err)
			return
		}
		p.ID, status = id, http.StatusOK
	}
	res, err := s.store.SpeicherePerson(p)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, status, res)
}

func (s *Server) loeschePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err == nil {
		err = s.store.LoeschePerson(id)
	}
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) ladeAusweisHoch(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAusweisBytes+1<<10)
	datei, kopf, err := r.FormFile("datei")
	if err != nil {
		schreibeFehler(w, ungueltig("keine Datei im Feld 'datei' oder Datei größer als 5 MB"))
		return
	}
	defer datei.Close()
	daten, err := io.ReadAll(io.LimitReader(datei, maxAusweisBytes+1))
	if err != nil {
		schreibeFehler(w, fmt.Errorf("Datei lesen: %w", err))
		return
	}
	if len(daten) > maxAusweisBytes {
		schreibeFehler(w, ungueltig("die Datei ist größer als 5 MB"))
		return
	}
	typ := http.DetectContentType(daten)
	if !erlaubteAusweisTypen[typ] {
		schreibeFehler(w, ungueltig("nur JPG, PNG oder PDF erlaubt (erkannt: %s)", typ))
		return
	}
	p, err := s.store.SpeichereAusweisDatei(id, filepath.Base(kopf.Filename), typ, daten)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, http.StatusOK, p)
}

func (s *Server) zeigeAusweis(w http.ResponseWriter, r *http.Request) {
	id, err := pfadID(r)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	a, err := s.store.AusweisDatei(id)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	w.Header().Set("Content-Type", a.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", a.Dateiname))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(a.Daten)
}

func (s *Server) listeAusleihen(w http.ResponseWriter, r *http.Request) {
	schreibeJSON(w, http.StatusOK, s.store.Ausleihen(s.now()))
}

type ausleiheAnfrage struct {
	SkiID    int       `json:"skiId"`
	PersonID int       `json:"personId"`
	Beginn   time.Time `json:"beginn"`
	Ende     time.Time `json:"ende"`
}

func (s *Server) neueAusleihe(w http.ResponseWriter, r *http.Request) {
	var req ausleiheAnfrage
	if err := liesJSON(r, &req); err != nil {
		schreibeFehler(w, err)
		return
	}
	res, err := s.store.NeueAusleihe(req.SkiID, req.PersonID, req.Beginn, req.Ende, s.now())
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, http.StatusCreated, res)
}

func (s *Server) rueckgabe(w http.ResponseWriter, r *http.Request) {
	s.aendereAusleihe(w, r, s.store.Rueckgabe)
}

func (s *Server) storno(w http.ResponseWriter, r *http.Request) {
	s.aendereAusleihe(w, r, s.store.Storno)
}

func (s *Server) aendereAusleihe(w http.ResponseWriter, r *http.Request, f func(int, time.Time) (store.AusleiheAnsicht, error)) {
	id, err := pfadID(r)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	res, err := f(id, s.now())
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, http.StatusOK, res)
}

func (s *Server) preise(w http.ResponseWriter, r *http.Request) {
	schreibeJSON(w, http.StatusOK, s.store.Preise())
}

func (s *Server) setzePreise(w http.ResponseWriter, r *http.Request) {
	var p domain.Preisliste
	if err := liesJSON(r, &p); err != nil {
		schreibeFehler(w, err)
		return
	}
	if err := s.store.SetzePreise(p); err != nil {
		schreibeFehler(w, err)
		return
	}
	schreibeJSON(w, http.StatusOK, s.store.Preise())
}

func (s *Server) berechnePreis(w http.ResponseWriter, r *http.Request) {
	von, bis, err := zeitraum(r)
	if err != nil {
		schreibeFehler(w, err)
		return
	}
	kat := r.URL.Query().Get("kategorie")
	tarif, ok := s.store.Preise()[kat]
	if !ok {
		schreibeFehler(w, ungueltig("unbekannte Kategorie %q", kat))
		return
	}
	preis, err := tarif.Berechne(von, bis)
	if err != nil {
		schreibeFehler(w, ungueltig("%v", err))
		return
	}
	stunden, _ := domain.Stunden(von, bis)
	schreibeJSON(w, http.StatusOK, map[string]int{"preisCent": preis, "stunden": stunden})
}
