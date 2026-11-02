package store

import (
	"time"

	"skiverleih/internal/domain"
)

// Seed füllt den Store mit Beispieldaten: 20 Ski, 10 Personen, 3 laufende Ausleihen, 2 Reservierungen.
func (s *Store) Seed(now time.Time) {
	ski := []domain.Ski{
		{Marke: "Atomic", Modell: "Redster X7", Kategorie: "Alpin", LaengeCm: 170},
		{Marke: "Atomic", Modell: "Redster X7", Kategorie: "Alpin", LaengeCm: 177},
		{Marke: "Fischer", Modell: "RC4 The Curv", Kategorie: "Alpin", LaengeCm: 165},
		{Marke: "Fischer", Modell: "RC4 The Curv", Kategorie: "Alpin", LaengeCm: 175},
		{Marke: "Head", Modell: "Supershape e-Magnum", Kategorie: "Alpin", LaengeCm: 163},
		{Marke: "Head", Modell: "Supershape e-Magnum", Kategorie: "Alpin", LaengeCm: 170},
		{Marke: "Rossignol", Modell: "Hero Elite", Kategorie: "Alpin", LaengeCm: 167},
		{Marke: "Völkl", Modell: "Deacon 76", Kategorie: "Alpin", LaengeCm: 158},
		{Marke: "Blizzard", Modell: "Rustler 10", Kategorie: "Freeride", LaengeCm: 180},
		{Marke: "Salomon", Modell: "QST 106", Kategorie: "Freeride", LaengeCm: 181},
		{Marke: "Faction", Modell: "Agent 3.0", Kategorie: "Freeride", LaengeCm: 172},
		{Marke: "Fischer", Modell: "Twin Skin Pro", Kategorie: "Langlauf", LaengeCm: 192},
		{Marke: "Fischer", Modell: "Twin Skin Pro", Kategorie: "Langlauf", LaengeCm: 197},
		{Marke: "Madshus", Modell: "Race Speed", Kategorie: "Langlauf", LaengeCm: 187},
		{Marke: "Dynafit", Modell: "Blacklight 88", Kategorie: "Touren", LaengeCm: 170},
		{Marke: "Ski Trab", Modell: "Maestro.2", Kategorie: "Touren", LaengeCm: 164},
		{Marke: "Atomic", Modell: "Redster J4", Kategorie: "Kinder", LaengeCm: 110},
		{Marke: "Head", Modell: "Supershape Team", Kategorie: "Kinder", LaengeCm: 117},
		{Marke: "Rossignol", Modell: "Hero Jr", Kategorie: "Kinder", LaengeCm: 100},
		{Marke: "Völkl", Modell: "Racetiger Jr", Kategorie: "Kinder", LaengeCm: 90},
	}
	for _, x := range ski {
		s.SpeichereSki(x)
	}

	personen := []domain.Person{
		{Vorname: "Anna", Nachname: "Berger", Email: "anna.berger@example.com", Telefon: "+43 660 1234567", Ausweis: domain.Ausweis{Typ: "Personalausweis", Nummer: "A1234567"}},
		{Vorname: "Lukas", Nachname: "Huber", Email: "lukas.huber@example.com", Ausweis: domain.Ausweis{Typ: "Reisepass", Nummer: "P9876543"}},
		{Vorname: "Sophie", Nachname: "Wagner", Email: "sophie.wagner@example.com", Ausweis: domain.Ausweis{Typ: "Führerschein", Nummer: "F55501"}},
		{Vorname: "Maximilian", Nachname: "Gruber", Telefon: "+43 664 7654321", Ausweis: domain.Ausweis{Typ: "Personalausweis", Nummer: "A7654321"}},
		{Vorname: "Laura", Nachname: "Pichler", Email: "laura.pichler@example.com", Ausweis: domain.Ausweis{Typ: "Reisepass", Nummer: "P1112223"}},
		{Vorname: "Jonas", Nachname: "Steiner", Ausweis: domain.Ausweis{Typ: "Personalausweis", Nummer: "A3332221"}},
		{Vorname: "Lena", Nachname: "Moser", Email: "lena.moser@example.com", Ausweis: domain.Ausweis{Typ: "Führerschein", Nummer: "F12890"}},
		{Vorname: "Felix", Nachname: "Hofer", Telefon: "+49 151 2223334", Ausweis: domain.Ausweis{Typ: "Reisepass", Nummer: "C4F7XK21"}},
		{Vorname: "Emma", Nachname: "Leitner", Email: "emma.leitner@example.com", Ausweis: domain.Ausweis{Typ: "Personalausweis", Nummer: "A9090901"}},
		{Vorname: "David", Nachname: "Bauer", Email: "david.bauer@example.com"}, // noch ohne Ausweis
	}
	ids := make([]int, len(personen))
	for i, p := range personen {
		gespeichert, _ := s.SpeicherePerson(p)
		ids[i] = gespeichert.ID
	}

	skiIDs := make([]int, 0, len(s.ski))
	for _, x := range s.Ski() {
		skiIDs = append(skiIDs, x.ID)
	}
	stunde := now.Truncate(time.Hour)
	buchungen := []struct {
		ski, person  int
		beginn, ende time.Time
	}{
		{skiIDs[0], ids[0], stunde.Add(-2 * time.Hour), stunde.Add(3 * time.Hour)},
		{skiIDs[8], ids[1], stunde.Add(-1 * time.Hour), stunde.Add(23 * time.Hour)},
		{skiIDs[16], ids[2], stunde.Add(-26 * time.Hour), stunde.Add(46 * time.Hour)},
		{skiIDs[11], ids[3], stunde.Add(26 * time.Hour), stunde.Add(30 * time.Hour)},
		{skiIDs[14], ids[4], stunde.Add(48 * time.Hour), stunde.Add(120 * time.Hour)},
	}
	for _, b := range buchungen {
		ski := s.ski[b.ski]
		a, _ := domain.NeueAusleihe(b.ski, b.person, b.beginn, b.ende, s.preise[ski.Kategorie])
		a.ID = s.neueID()
		s.ausleihen[a.ID] = a
	}
}
