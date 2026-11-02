package domain

import (
	"errors"
	"time"
)

// Status einer Ausleihe; er wird aus den Zeitpunkten abgeleitet und nicht gespeichert.
type Status string

const (
	StatusReserviert    Status = "reserviert"
	StatusAktiv         Status = "aktiv"
	StatusUeberfaellig  Status = "überfällig"
	StatusAbgeschlossen Status = "abgeschlossen"
	StatusStorniert     Status = "storniert"
)

// Ausleihe verbindet einen Ski mit einer Person für einen Zeitraum.
// Liegt der Beginn in der Zukunft, ist es eine Reservierung.
type Ausleihe struct {
	ID           int        `json:"id"`
	SkiID        int        `json:"skiId"`
	PersonID     int        `json:"personId"`
	Beginn       time.Time  `json:"beginn"`
	Ende         time.Time  `json:"ende"`
	Tarif        Tarif      `json:"tarif"`
	PreisCent    int        `json:"preisCent"`
	Rueckgabe    *time.Time `json:"rueckgabe,omitempty"`
	EndpreisCent int        `json:"endpreisCent,omitempty"`
	Storniert    bool       `json:"storniert"`
}

// NeueAusleihe legt eine Ausleihe an und berechnet den vereinbarten Preis.
func NeueAusleihe(skiID, personID int, beginn, ende time.Time, tarif Tarif) (Ausleihe, error) {
	preis, err := tarif.Berechne(beginn, ende)
	if err != nil {
		return Ausleihe{}, err
	}
	return Ausleihe{SkiID: skiID, PersonID: personID, Beginn: beginn, Ende: ende, Tarif: tarif, PreisCent: preis}, nil
}

// Status liefert den Zustand der Ausleihe zum Zeitpunkt now.
func (a Ausleihe) Status(now time.Time) Status {
	switch {
	case a.Storniert:
		return StatusStorniert
	case a.Rueckgabe != nil:
		return StatusAbgeschlossen
	case now.Before(a.Beginn):
		return StatusReserviert
	case now.After(a.Ende):
		return StatusUeberfaellig
	default:
		return StatusAktiv
	}
}

// Offen meldet, ob die Ausleihe den Ski (jetzt oder künftig) belegt.
func (a Ausleihe) Offen() bool {
	return !a.Storniert && a.Rueckgabe == nil
}

// Unterwegs meldet, ob der Ski gerade beim Kunden ist (aktiv oder überfällig).
func (a Ausleihe) Unterwegs(now time.Time) bool {
	s := a.Status(now)
	return s == StatusAktiv || s == StatusUeberfaellig
}

// Ueberschneidet prüft, ob die Ausleihe den Zeitraum [von, bis) blockiert.
// Ein überfälliger Ski bleibt bis zu seiner Rückgabe auf unbestimmte Zeit blockiert.
func (a Ausleihe) Ueberschneidet(von, bis, now time.Time) bool {
	if !a.Offen() {
		return false
	}
	if a.Status(now) == StatusUeberfaellig {
		return bis.After(a.Beginn)
	}
	return von.Before(a.Ende) && a.Beginn.Before(bis)
}

// Zurueckgeben schließt die Ausleihe ab und rechnet nach tatsächlicher Dauer ab.
func (a *Ausleihe) Zurueckgeben(now time.Time) error {
	if !a.Unterwegs(now) {
		return errors.New("nur aktive oder überfällige Ausleihen können zurückgegeben werden")
	}
	preis, err := a.Tarif.Berechne(a.Beginn, now)
	if err != nil {
		return err
	}
	a.Rueckgabe = &now
	a.EndpreisCent = preis
	return nil
}

// Stornieren storniert eine noch nicht begonnene Reservierung.
func (a *Ausleihe) Stornieren(now time.Time) error {
	if a.Status(now) != StatusReserviert {
		return errors.New("nur Reservierungen können storniert werden")
	}
	a.Storniert = true
	return nil
}
