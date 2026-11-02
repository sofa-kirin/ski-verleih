package store

import (
	"errors"
	"sync"
	"testing"
	"time"

	"skiverleih/internal/domain"
)

var now = time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)

func setup(t *testing.T) (*Store, domain.Ski, domain.Person) {
	t.Helper()
	s := New()
	ski, err := s.SpeichereSki(domain.Ski{Marke: "Atomic", Modell: "X", Kategorie: "Alpin", LaengeCm: 170})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.SpeicherePerson(domain.Person{Vorname: "Anna", Nachname: "B", Ausweis: domain.Ausweis{Typ: "Reisepass", Nummer: "p1"}})
	if err != nil {
		t.Fatal(err)
	}
	return s, ski, p
}

func TestNeueAusleiheKonflikt(t *testing.T) {
	s, ski, p := setup(t)
	if _, err := s.NeueAusleihe(ski.ID, p.ID, now, now.Add(4*time.Hour), now); err != nil {
		t.Fatal(err)
	}
	_, err := s.NeueAusleihe(ski.ID, p.ID, now.Add(2*time.Hour), now.Add(6*time.Hour), now)
	if !errors.Is(err, ErrKonflikt) {
		t.Errorf("erwartet Konflikt, got %v", err)
	}
	if _, err := s.NeueAusleihe(ski.ID, p.ID, now.Add(4*time.Hour), now.Add(6*time.Hour), now); err != nil {
		t.Errorf("anschließende Reservierung abgelehnt: %v", err)
	}
	frei, _ := s.Verfuegbar(now.Add(time.Hour), now.Add(2*time.Hour), now)
	if len(frei) != 0 {
		t.Errorf("Ski sollte nicht verfügbar sein: %v", frei)
	}
}

func TestNeueAusleiheBrauchtAusweis(t *testing.T) {
	s, ski, _ := setup(t)
	ohne, _ := s.SpeicherePerson(domain.Person{Vorname: "Ohne", Nachname: "Ausweis"})
	_, err := s.NeueAusleihe(ski.ID, ohne.ID, now, now.Add(time.Hour), now)
	if !errors.Is(err, ErrUngueltig) {
		t.Fatalf("erwartet ErrUngueltig, got %v", err)
	}
	if _, err := s.SpeichereAusweisDatei(ohne.ID, "pass.jpg", "image/jpeg", []byte{1, 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.NeueAusleihe(ski.ID, ohne.ID, now, now.Add(time.Hour), now); err != nil {
		t.Errorf("mit hochgeladenem Ausweis muss es gehen: %v", err)
	}
}

func TestAusweisDateiBleibtBeimBearbeiten(t *testing.T) {
	s, _, p := setup(t)
	s.SpeichereAusweisDatei(p.ID, "pass.pdf", "application/pdf", []byte("x"))
	p.Telefon = "123"
	s.SpeicherePerson(p)
	if a, err := s.AusweisDatei(p.ID); err != nil || a.Dateiname != "pass.pdf" {
		t.Errorf("Datei verloren: %v", err)
	}
}

func TestBeginnInVergangenheit(t *testing.T) {
	s, ski, p := setup(t)
	_, err := s.NeueAusleihe(ski.ID, p.ID, now.Add(-2*time.Hour), now.Add(time.Hour), now)
	if !errors.Is(err, ErrUngueltig) {
		t.Errorf("got %v", err)
	}
}

func TestDashboardUndRueckgabe(t *testing.T) {
	s, ski, p := setup(t)
	s.SpeichereSki(domain.Ski{Marke: "Head", Modell: "Y", Kategorie: "Kinder", LaengeCm: 100})
	a, _ := s.NeueAusleihe(ski.ID, p.ID, now, now.Add(4*time.Hour), now)
	s.NeueAusleihe(ski.ID, p.ID, now.Add(24*time.Hour), now.Add(26*time.Hour), now)

	d := s.Dashboard(now.Add(time.Hour))
	if d.GesamtSki != 2 || d.AusgelieheneSki != 1 || d.VerfuegbareSki != 1 || d.Reservierungen != 1 || d.Personen != 1 {
		t.Errorf("Dashboard falsch: %+v", d)
	}
	if err := s.LoescheSki(ski.ID); !errors.Is(err, ErrKonflikt) {
		t.Errorf("Löschen mit offener Ausleihe muss scheitern: %v", err)
	}
	r, err := s.Rueckgabe(a.ID, now.Add(90*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if r.EndpreisCent != 1200 || r.Status != domain.StatusAbgeschlossen {
		t.Errorf("Rückgabe falsch: %+v", r)
	}
	if d := s.Dashboard(now.Add(2 * time.Hour)); d.AusgelieheneSki != 0 {
		t.Errorf("nach Rückgabe noch ausgeliehen: %+v", d)
	}
}

func TestPreiseGeltenNurFuerNeueBuchungen(t *testing.T) {
	s, ski, p := setup(t)
	a, _ := s.NeueAusleihe(ski.ID, p.ID, now, now.Add(2*time.Hour), now)
	neu := s.Preise()
	neu["Alpin"] = domain.Tarif{StundenpreisCent: 1000, TagespreisCent: 5000}
	if err := s.SetzePreise(neu); err != nil {
		t.Fatal(err)
	}
	r, _ := s.Rueckgabe(a.ID, now.Add(2*time.Hour))
	if r.EndpreisCent != 1200 {
		t.Errorf("alter Tarif muss gelten, Endpreis %d", r.EndpreisCent)
	}
	delete(neu, "Alpin")
	if err := s.SetzePreise(neu); !errors.Is(err, ErrKonflikt) {
		t.Errorf("verwendete Kategorie darf nicht gelöscht werden: %v", err)
	}
}

func TestParalleleBuchungen(t *testing.T) {
	s, ski, p := setup(t)
	var wg sync.WaitGroup
	erfolge := make(chan struct{}, 50)
	for range 50 {
		wg.Go(func() {
			if _, err := s.NeueAusleihe(ski.ID, p.ID, now, now.Add(time.Hour), now); err == nil {
				erfolge <- struct{}{}
			}
		})
	}
	wg.Wait()
	if len(erfolge) != 1 {
		t.Errorf("genau eine Buchung erwartet, got %d", len(erfolge))
	}
}

func TestSeed(t *testing.T) {
	s := New()
	s.Seed(now)
	d := s.Dashboard(now)
	if d.GesamtSki != 20 || d.AusgelieheneSki != 3 || d.VerfuegbareSki != 17 || d.Personen != 10 || d.Reservierungen != 2 {
		t.Errorf("Seed-Dashboard: %+v", d)
	}
}
