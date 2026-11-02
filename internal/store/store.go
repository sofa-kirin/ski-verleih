// Package store hält alle Daten thread-safe im Speicher.
package store

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"skiverleih/internal/domain"
)

var (
	ErrNichtGefunden = errors.New("nicht gefunden")
	ErrUngueltig     = errors.New("ungültige Eingabe")
	ErrKonflikt      = errors.New("Konflikt")
)

// Store ist der In-Memory-Datenspeicher des Verleihs.
type Store struct {
	mu        sync.RWMutex
	ski       map[int]domain.Ski
	personen  map[int]domain.Person
	ausleihen map[int]domain.Ausleihe
	preise    domain.Preisliste
	nextID    int
}

// New erzeugt einen leeren Store mit der Standard-Preisliste.
func New() *Store {
	return &Store{
		ski:       map[int]domain.Ski{},
		personen:  map[int]domain.Person{},
		ausleihen: map[int]domain.Ausleihe{},
		preise:    domain.StandardPreisliste(),
	}
}

func (s *Store) neueID() int {
	s.nextID++
	return s.nextID
}

func ungueltig(err error) error {
	return fmt.Errorf("%w: %w", ErrUngueltig, err)
}

func sortiert[T any](m map[int]T, less func(a, b T) int) []T {
	return slices.SortedFunc(maps.Values(m), less)
}

// ---------- Ski ----------

// Ski liefert alle Ski, sortiert nach ID.
func (s *Store) Ski() []domain.Ski {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortiert(s.ski, func(a, b domain.Ski) int { return cmp.Compare(a.ID, b.ID) })
}

// SpeichereSki legt einen Ski an (ID 0) oder ändert ihn.
func (s *Store) SpeichereSki(ski domain.Ski) (domain.Ski, error) {
	ski.Normalisieren()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ski.Validate(s.preise); err != nil {
		return domain.Ski{}, ungueltig(err)
	}
	if ski.ID == 0 {
		ski.ID = s.neueID()
	} else if _, ok := s.ski[ski.ID]; !ok {
		return domain.Ski{}, fmt.Errorf("Ski %d: %w", ski.ID, ErrNichtGefunden)
	}
	s.ski[ski.ID] = ski
	return ski, nil
}

// LoescheSki entfernt einen Ski ohne offene Ausleihen.
func (s *Store) LoescheSki(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.ski[id]; !ok {
		return fmt.Errorf("Ski %d: %w", id, ErrNichtGefunden)
	}
	for _, a := range s.ausleihen {
		if a.SkiID == id && a.Offen() {
			return fmt.Errorf("%w: der Ski hat noch offene Ausleihen", ErrKonflikt)
		}
	}
	delete(s.ski, id)
	return nil
}

// Angebot ist ein im Zeitraum freier Ski mit seinem Preis.
type Angebot struct {
	domain.Ski
	PreisCent int `json:"preisCent"`
}

// Verfuegbar liefert alle Ski, die im Zeitraum [von, bis) frei sind, inkl. Preis.
func (s *Store) Verfuegbar(von, bis, now time.Time) ([]Angebot, error) {
	if !bis.After(von) {
		return nil, ungueltig(domain.ErrUngueltigerZeitraum)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []Angebot
	for _, ski := range s.ski {
		if s.belegt(ski.ID, von, bis, now) {
			continue
		}
		preis, _ := s.preise[ski.Kategorie].Berechne(von, bis)
		res = append(res, Angebot{Ski: ski, PreisCent: preis})
	}
	slices.SortFunc(res, func(a, b Angebot) int {
		return cmp.Or(cmp.Compare(a.Kategorie, b.Kategorie), cmp.Compare(a.ID, b.ID))
	})
	return res, nil
}

func (s *Store) belegt(skiID int, von, bis, now time.Time) bool {
	for _, a := range s.ausleihen {
		if a.SkiID == skiID && a.Ueberschneidet(von, bis, now) {
			return true
		}
	}
	return false
}

// ---------- Personen ----------

// Personen liefert alle Personen, sortiert nach Nachname.
func (s *Store) Personen() []domain.Person {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortiert(s.personen, func(a, b domain.Person) int {
		return cmp.Or(cmp.Compare(a.Nachname, b.Nachname), cmp.Compare(a.Vorname, b.Vorname))
	})
}

// SpeicherePerson legt eine Person an (ID 0) oder ändert sie.
// Eine bereits hochgeladene Ausweis-Datei bleibt beim Ändern erhalten.
func (s *Store) SpeicherePerson(p domain.Person) (domain.Person, error) {
	p.Normalisieren()
	if err := p.Validate(); err != nil {
		return domain.Person{}, ungueltig(err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == 0 {
		p.ID = s.neueID()
		p.Ausweis.Dateiname, p.Ausweis.ContentType, p.Ausweis.Daten = "", "", nil
	} else {
		alt, ok := s.personen[p.ID]
		if !ok {
			return domain.Person{}, fmt.Errorf("Person %d: %w", p.ID, ErrNichtGefunden)
		}
		p.Ausweis.Dateiname, p.Ausweis.ContentType, p.Ausweis.Daten = alt.Ausweis.Dateiname, alt.Ausweis.ContentType, alt.Ausweis.Daten
	}
	s.personen[p.ID] = p
	return p, nil
}

// SpeichereAusweisDatei hinterlegt einen Foto/Scan des Ausweises.
func (s *Store) SpeichereAusweisDatei(id int, name, contentType string, daten []byte) (domain.Person, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.personen[id]
	if !ok {
		return domain.Person{}, fmt.Errorf("Person %d: %w", id, ErrNichtGefunden)
	}
	p.Ausweis.Dateiname, p.Ausweis.ContentType, p.Ausweis.Daten = name, contentType, daten
	s.personen[id] = p
	return p, nil
}

// AusweisDatei liefert die hochgeladene Ausweis-Datei einer Person.
func (s *Store) AusweisDatei(id int) (domain.Ausweis, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.personen[id]
	if !ok || len(p.Ausweis.Daten) == 0 {
		return domain.Ausweis{}, fmt.Errorf("Ausweis-Datei: %w", ErrNichtGefunden)
	}
	return p.Ausweis, nil
}

// LoeschePerson entfernt eine Person ohne offene Ausleihen.
func (s *Store) LoeschePerson(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.personen[id]; !ok {
		return fmt.Errorf("Person %d: %w", id, ErrNichtGefunden)
	}
	for _, a := range s.ausleihen {
		if a.PersonID == id && a.Offen() {
			return fmt.Errorf("%w: die Person hat noch offene Ausleihen", ErrKonflikt)
		}
	}
	delete(s.personen, id)
	return nil
}

// ---------- Ausleihen ----------

// AusleiheAnsicht ergänzt eine Ausleihe um Namen und aktuellen Status für die Anzeige.
type AusleiheAnsicht struct {
	domain.Ausleihe
	Ski    string        `json:"ski"`
	Person string        `json:"person"`
	Status domain.Status `json:"status"`
}

func (s *Store) ansicht(a domain.Ausleihe, now time.Time) AusleiheAnsicht {
	ski, person := "(gelöscht)", "(gelöscht)"
	if x, ok := s.ski[a.SkiID]; ok {
		ski = x.Bezeichnung()
	}
	if x, ok := s.personen[a.PersonID]; ok {
		person = x.Name()
	}
	return AusleiheAnsicht{Ausleihe: a, Ski: ski, Person: person, Status: a.Status(now)}
}

// Ausleihen liefert alle Ausleihen, neueste zuerst.
func (s *Store) Ausleihen(now time.Time) []AusleiheAnsicht {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]AusleiheAnsicht, 0, len(s.ausleihen))
	for _, a := range s.ausleihen {
		res = append(res, s.ansicht(a, now))
	}
	slices.SortFunc(res, func(a, b AusleiheAnsicht) int { return b.Beginn.Compare(a.Beginn) })
	return res
}

// NeueAusleihe bucht einen Ski für eine Person im Zeitraum beginn–ende.
func (s *Store) NeueAusleihe(skiID, personID int, beginn, ende, now time.Time) (AusleiheAnsicht, error) {
	if beginn.Before(now.Add(-time.Hour)) {
		return AusleiheAnsicht{}, ungueltig(errors.New("der Beginn darf höchstens 1 Stunde in der Vergangenheit liegen"))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ski, ok := s.ski[skiID]
	if !ok {
		return AusleiheAnsicht{}, fmt.Errorf("Ski %d: %w", skiID, ErrNichtGefunden)
	}
	p, ok := s.personen[personID]
	if !ok {
		return AusleiheAnsicht{}, fmt.Errorf("Person %d: %w", personID, ErrNichtGefunden)
	}
	if !p.Ausweis.Hinterlegt() {
		return AusleiheAnsicht{}, ungueltig(fmt.Errorf("für %s ist kein Ausweis hinterlegt", p.Name()))
	}
	a, err := domain.NeueAusleihe(skiID, personID, beginn, ende, s.preise[ski.Kategorie])
	if err != nil {
		return AusleiheAnsicht{}, ungueltig(err)
	}
	if s.belegt(skiID, beginn, ende, now) {
		return AusleiheAnsicht{}, fmt.Errorf("%w: %s ist in diesem Zeitraum bereits vergeben", ErrKonflikt, ski.Bezeichnung())
	}
	a.ID = s.neueID()
	s.ausleihen[a.ID] = a
	return s.ansicht(a, now), nil
}

// Rueckgabe schließt eine Ausleihe ab.
func (s *Store) Rueckgabe(id int, now time.Time) (AusleiheAnsicht, error) {
	return s.aendereAusleihe(id, now, func(a *domain.Ausleihe) error { return a.Zurueckgeben(now) })
}

// Storno storniert eine Reservierung.
func (s *Store) Storno(id int, now time.Time) (AusleiheAnsicht, error) {
	return s.aendereAusleihe(id, now, func(a *domain.Ausleihe) error { return a.Stornieren(now) })
}

func (s *Store) aendereAusleihe(id int, now time.Time, f func(*domain.Ausleihe) error) (AusleiheAnsicht, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.ausleihen[id]
	if !ok {
		return AusleiheAnsicht{}, fmt.Errorf("Ausleihe %d: %w", id, ErrNichtGefunden)
	}
	if err := f(&a); err != nil {
		return AusleiheAnsicht{}, fmt.Errorf("%w: %w", ErrKonflikt, err)
	}
	s.ausleihen[id] = a
	return s.ansicht(a, now), nil
}

// ---------- Preise ----------

// Preise liefert eine Kopie der aktuellen Preisliste.
func (s *Store) Preise() domain.Preisliste {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.preise)
}

// SetzePreise ersetzt die Tarife. Kategorien, die noch von Ski verwendet werden, müssen erhalten bleiben.
func (s *Store) SetzePreise(p domain.Preisliste) error {
	if err := p.Validate(); err != nil {
		return ungueltig(err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ski := range s.ski {
		if _, ok := p[ski.Kategorie]; !ok {
			return fmt.Errorf("%w: Kategorie %q wird noch verwendet", ErrKonflikt, ski.Kategorie)
		}
	}
	s.preise = maps.Clone(p)
	return nil
}

// ---------- Dashboard ----------

// Dashboard enthält die Kennzahlen der Startseite.
type Dashboard struct {
	VerfuegbareSki  int               `json:"verfuegbareSki"`
	AusgelieheneSki int               `json:"ausgelieheneSki"`
	GesamtSki       int               `json:"gesamtSki"`
	Personen        int               `json:"personen"`
	Ueberfaellig    int               `json:"ueberfaellig"`
	Reservierungen  int               `json:"reservierungen"`
	Offen           []AusleiheAnsicht `json:"offen"`
}

// Dashboard berechnet die Kennzahlen zum Zeitpunkt now.
func (s *Store) Dashboard(now time.Time) Dashboard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d := Dashboard{GesamtSki: len(s.ski), Personen: len(s.personen), Offen: []AusleiheAnsicht{}}
	unterwegs := map[int]bool{}
	for _, a := range s.ausleihen {
		if !a.Offen() {
			continue
		}
		switch a.Status(now) {
		case domain.StatusReserviert:
			d.Reservierungen++
		case domain.StatusUeberfaellig:
			d.Ueberfaellig++
			unterwegs[a.SkiID] = true
		case domain.StatusAktiv:
			unterwegs[a.SkiID] = true
		}
		d.Offen = append(d.Offen, s.ansicht(a, now))
	}
	d.AusgelieheneSki = len(unterwegs)
	d.VerfuegbareSki = d.GesamtSki - d.AusgelieheneSki
	slices.SortFunc(d.Offen, func(a, b AusleiheAnsicht) int {
		// überfällig zuerst, dann nach nächstem relevanten Zeitpunkt
		return cmp.Or(cmp.Compare(rang(a.Status), rang(b.Status)), a.Ende.Compare(b.Ende))
	})
	return d
}

func rang(s domain.Status) int {
	switch s {
	case domain.StatusUeberfaellig:
		return 0
	case domain.StatusAktiv:
		return 1
	default:
		return 2
	}
}
