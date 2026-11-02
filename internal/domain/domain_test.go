package domain

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)

func TestTarifBerechne(t *testing.T) {
	alpin := Tarif{StundenpreisCent: 600, TagespreisCent: 3000}
	tests := []struct {
		name  string
		dauer time.Duration
		want  int
	}{
		{"angefangene Stunde zählt voll", 10 * time.Minute, 600},
		{"3 Stunden", 3 * time.Hour, 1800},
		{"Tagesdeckel greift", 8 * time.Hour, 3000},
		{"genau ein Tag", 24 * time.Hour, 3000},
		{"Tag plus 2 Stunden", 26 * time.Hour, 4200},
		{"drei Tage", 72 * time.Hour, 9000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := alpin.Berechne(t0, t0.Add(tt.dauer))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
	if _, err := alpin.Berechne(t0, t0); err == nil {
		t.Error("leerer Zeitraum muss abgelehnt werden")
	}
}

func TestTarifValidate(t *testing.T) {
	if err := (Tarif{StundenpreisCent: 500, TagespreisCent: 400}).Validate(); err == nil {
		t.Error("Tagespreis < Stundenpreis muss abgelehnt werden")
	}
	if err := (Tarif{StundenpreisCent: 0, TagespreisCent: 400}).Validate(); err == nil {
		t.Error("Preis 0 muss abgelehnt werden")
	}
}

func TestPersonValidateAusweis(t *testing.T) {
	p := Person{Vorname: "Anna", Nachname: "Berger", Ausweis: Ausweis{Typ: "Reisepass"}}
	if err := p.Validate(); err == nil {
		t.Error("Typ ohne Nummer muss abgelehnt werden")
	}
	p.Ausweis = Ausweis{Typ: "Bibliotheksausweis", Nummer: "1"}
	if err := p.Validate(); err == nil {
		t.Error("unbekannter Typ muss abgelehnt werden")
	}
	p.Ausweis = Ausweis{Typ: "Reisepass", Nummer: "P123"}
	if err := p.Validate(); err != nil || !p.Ausweis.Hinterlegt() {
		t.Errorf("gültiger Ausweis abgelehnt: %v", err)
	}
	if (Ausweis{Daten: []byte{1}}).Hinterlegt() == false {
		t.Error("hochgeladene Datei muss als Ausweis reichen")
	}
}

func TestAusleiheStatusUndRueckgabe(t *testing.T) {
	tarif := Tarif{StundenpreisCent: 600, TagespreisCent: 3000}
	a, err := NeueAusleihe(1, 1, t0, t0.Add(4*time.Hour), tarif)
	if err != nil {
		t.Fatal(err)
	}
	if a.PreisCent != 2400 {
		t.Errorf("Preis %d, want 2400", a.PreisCent)
	}
	if s := a.Status(t0.Add(-time.Hour)); s != StatusReserviert {
		t.Errorf("Status vor Beginn = %s", s)
	}
	if s := a.Status(t0.Add(time.Hour)); s != StatusAktiv {
		t.Errorf("Status während = %s", s)
	}
	if s := a.Status(t0.Add(5 * time.Hour)); s != StatusUeberfaellig {
		t.Errorf("Status nach Ende = %s", s)
	}
	if err := a.Zurueckgeben(t0.Add(-time.Hour)); err == nil {
		t.Error("Rückgabe vor Beginn muss abgelehnt werden")
	}
	// frühere Rückgabe nach 2 h 10 min → 3 angefangene Stunden
	if err := a.Zurueckgeben(t0.Add(2*time.Hour + 10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if a.EndpreisCent != 1800 || a.Status(t0) != StatusAbgeschlossen {
		t.Errorf("Endpreis %d, Status %s", a.EndpreisCent, a.Status(t0))
	}
}

func TestAusleiheUeberschneidet(t *testing.T) {
	a, _ := NeueAusleihe(1, 1, t0, t0.Add(4*time.Hour), Tarif{StundenpreisCent: 1, TagespreisCent: 1})
	now := t0.Add(-time.Hour)
	if !a.Ueberschneidet(t0.Add(time.Hour), t0.Add(5*time.Hour), now) {
		t.Error("überlappender Zeitraum nicht erkannt")
	}
	if a.Ueberschneidet(t0.Add(4*time.Hour), t0.Add(6*time.Hour), now) {
		t.Error("direkt anschließender Zeitraum darf nicht kollidieren")
	}
	// überfällig: blockiert auch spätere Zeiträume
	if !a.Ueberschneidet(t0.Add(48*time.Hour), t0.Add(50*time.Hour), t0.Add(5*time.Hour)) {
		t.Error("überfälliger Ski muss blockiert bleiben")
	}
	if err := a.Stornieren(now); err != nil || a.Ueberschneidet(t0, t0.Add(time.Hour), now) {
		t.Error("stornierte Reservierung darf nicht blockieren")
	}
}
