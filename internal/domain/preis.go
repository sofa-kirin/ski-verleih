package domain

import (
	"errors"
	"time"
)

// ErrUngueltigerZeitraum wird geliefert, wenn das Ende nicht nach dem Beginn liegt.
var ErrUngueltigerZeitraum = errors.New("das Ende muss nach dem Beginn liegen")

// Tarif beschreibt den Preis einer Ski-Kategorie.
// Abgerechnet wird pro angefangener Stunde, pro angefangenem 24-h-Block höchstens der Tagespreis.
type Tarif struct {
	StundenpreisCent int `json:"stundenpreisCent"`
	TagespreisCent   int `json:"tagespreisCent"`
}

// Validate prüft, ob der Tarif sinnvoll ist.
func (t Tarif) Validate() error {
	if t.StundenpreisCent <= 0 || t.TagespreisCent <= 0 {
		return errors.New("Preise müssen größer als 0 sein")
	}
	if t.TagespreisCent < t.StundenpreisCent {
		return errors.New("der Tagespreis darf nicht kleiner als der Stundenpreis sein")
	}
	return nil
}

// Berechne liefert den Preis in Cent für den Zeitraum beginn–ende.
func (t Tarif) Berechne(beginn, ende time.Time) (int, error) {
	h, err := Stunden(beginn, ende)
	if err != nil {
		return 0, err
	}
	tage, rest := h/24, h%24
	return tage*t.TagespreisCent + min(rest*t.StundenpreisCent, t.TagespreisCent), nil
}

// Stunden liefert die abzurechnenden Stunden; angefangene Stunden zählen voll.
func Stunden(beginn, ende time.Time) (int, error) {
	if !ende.After(beginn) {
		return 0, ErrUngueltigerZeitraum
	}
	d := ende.Sub(beginn)
	h := int(d / time.Hour)
	if d%time.Hour != 0 {
		h++
	}
	return h, nil
}

// Preisliste ordnet jeder Ski-Kategorie einen Tarif zu.
type Preisliste map[string]Tarif

// StandardPreisliste liefert die Startpreise des Verleihs.
func StandardPreisliste() Preisliste {
	return Preisliste{
		"Alpin":    {StundenpreisCent: 600, TagespreisCent: 3000},
		"Freeride": {StundenpreisCent: 800, TagespreisCent: 4000},
		"Langlauf": {StundenpreisCent: 400, TagespreisCent: 1800},
		"Touren":   {StundenpreisCent: 700, TagespreisCent: 3500},
		"Kinder":   {StundenpreisCent: 300, TagespreisCent: 1500},
	}
}

// Validate prüft alle Tarife der Preisliste.
func (p Preisliste) Validate() error {
	if len(p) == 0 {
		return errors.New("die Preisliste ist leer")
	}
	for kat, t := range p {
		if kat == "" {
			return errors.New("Kategorie ohne Namen")
		}
		if err := t.Validate(); err != nil {
			return errors.New(kat + ": " + err.Error())
		}
	}
	return nil
}
