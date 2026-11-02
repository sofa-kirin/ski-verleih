package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Ausweistypen sind die akzeptierten Ausweisdokumente.
var Ausweistypen = []string{"Personalausweis", "Reisepass", "Führerschein"}

// Ausweis eines Kunden: eingegeben (Typ + Nummer) und/oder als Datei hochgeladen.
type Ausweis struct {
	Typ         string `json:"typ"`
	Nummer      string `json:"nummer"`
	Dateiname   string `json:"dateiname,omitempty"`
	ContentType string `json:"-"`
	Daten       []byte `json:"-"`
}

// Hinterlegt meldet, ob ein Ausweis vorliegt, der für eine Ausleihe ausreicht.
func (a Ausweis) Hinterlegt() bool {
	return (a.Typ != "" && a.Nummer != "") || len(a.Daten) > 0
}

// Person ist ein registrierter Kunde.
type Person struct {
	ID       int     `json:"id"`
	Vorname  string  `json:"vorname"`
	Nachname string  `json:"nachname"`
	Email    string  `json:"email"`
	Telefon  string  `json:"telefon"`
	Ausweis  Ausweis `json:"ausweis"`
}

// Name liefert "Vorname Nachname".
func (p Person) Name() string {
	return p.Vorname + " " + p.Nachname
}

// Normalisieren entfernt überflüssige Leerzeichen.
func (p *Person) Normalisieren() {
	p.Vorname = strings.TrimSpace(p.Vorname)
	p.Nachname = strings.TrimSpace(p.Nachname)
	p.Email = strings.TrimSpace(p.Email)
	p.Telefon = strings.TrimSpace(p.Telefon)
	p.Ausweis.Typ = strings.TrimSpace(p.Ausweis.Typ)
	p.Ausweis.Nummer = strings.ToUpper(strings.TrimSpace(p.Ausweis.Nummer))
}

// Validate prüft Pflichtfelder und Ausweisangaben.
func (p Person) Validate() error {
	if p.Vorname == "" || p.Nachname == "" {
		return errors.New("Vorname und Nachname sind Pflichtfelder")
	}
	if p.Email != "" && !strings.Contains(p.Email, "@") {
		return errors.New("die E-Mail-Adresse ist ungültig")
	}
	a := p.Ausweis
	if (a.Typ == "") != (a.Nummer == "") {
		return errors.New("Ausweistyp und Ausweisnummer bitte gemeinsam angeben")
	}
	if a.Typ != "" && !slices.Contains(Ausweistypen, a.Typ) {
		return fmt.Errorf("unbekannter Ausweistyp %q", a.Typ)
	}
	return nil
}
