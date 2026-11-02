package domain

import (
	"errors"
	"fmt"
	"strings"
)

// Ski ist ein verleihbares Paar Ski.
type Ski struct {
	ID        int    `json:"id"`
	Marke     string `json:"marke"`
	Modell    string `json:"modell"`
	Kategorie string `json:"kategorie"`
	LaengeCm  int    `json:"laengeCm"`
}

// Bezeichnung liefert einen lesbaren Namen, z.B. "Atomic Redster (170 cm)".
func (s Ski) Bezeichnung() string {
	return fmt.Sprintf("%s %s (%d cm)", s.Marke, s.Modell, s.LaengeCm)
}

// Normalisieren entfernt überflüssige Leerzeichen.
func (s *Ski) Normalisieren() {
	s.Marke = strings.TrimSpace(s.Marke)
	s.Modell = strings.TrimSpace(s.Modell)
	s.Kategorie = strings.TrimSpace(s.Kategorie)
}

// Validate prüft die Pflichtfelder; die Kategorie muss in der Preisliste existieren.
func (s Ski) Validate(preise Preisliste) error {
	if s.Marke == "" || s.Modell == "" {
		return errors.New("Marke und Modell sind Pflichtfelder")
	}
	if _, ok := preise[s.Kategorie]; !ok {
		return fmt.Errorf("unbekannte Kategorie %q", s.Kategorie)
	}
	if s.LaengeCm < 70 || s.LaengeCm > 220 {
		return errors.New("die Länge muss zwischen 70 und 220 cm liegen")
	}
	return nil
}
