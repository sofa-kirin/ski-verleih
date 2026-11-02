# Ski-Verleih

Prototyp eines Verleihsystems für Ski, entwickelt mit Spec-Driven Development (SDD) und Claude Code.
Dieses Dokument ist die **Spezifikation**; `CLAUDE.md` enthält die technischen Regeln für den KI-Assistenten.

Umfang: Aufgabe 1 a) (Ski-Verleih mit Dashboard und Ausweis) und b) (flexible Preise, Verleihdauer und Verleihbeginn).

## Starten

```bash
go run .
```

Dann http://localhost:8080 öffnen. Beim Start werden Beispieldaten geladen (20 Ski, 10 Personen, 3 laufende Ausleihen, 2 Reservierungen).

Tests: `go test ./...`

---

## 1. Fachliche Spezifikation

### 1.1 Ski
| Feld | Typ | Regel |
|---|---|---|
| `id` | int | vom System vergeben |
| `marke` | string | Pflicht |
| `modell` | string | Pflicht |
| `kategorie` | string | Pflicht, muss in der Preisliste existieren (Alpin, Freeride, Langlauf, Touren, Kinder) |
| `laengeCm` | int | 70 – 220 |

Ein Ski kann nur gelöscht werden, wenn keine offene Ausleihe/Reservierung auf ihn existiert.

### 1.2 Person (Kunde) mit Ausweis – *a)*
| Feld | Typ | Regel |
|---|---|---|
| `vorname`, `nachname` | string | Pflicht |
| `email`, `telefon` | string | optional |
| `ausweis.typ` | string | Personalausweis, Reisepass oder Führerschein |
| `ausweis.nummer` | string | Ausweisnummer |
| Ausweis-Datei | Upload | Foto/Scan (JPG, PNG, PDF), max. 5 MB, ersetzt eine vorhandene Datei |

- Der Ausweis kann **eingegeben** (Typ + Nummer) und/oder **hochgeladen** werden.
- Typ und Nummer werden nur gemeinsam akzeptiert (beides oder keines).
- **Eine Ausleihe ist nur möglich, wenn ein Ausweis hinterlegt ist** (Typ + Nummer oder Datei).
- Eine Person mit offener Ausleihe kann nicht gelöscht werden.

### 1.3 Preise – *b)*
Für jede Ski-Kategorie gibt es einen **Tarif**, der in der Oberfläche jederzeit änderbar ist:

| Feld | Bedeutung |
|---|---|
| `stundenpreisCent` | Preis pro angefangener Stunde |
| `tagespreisCent` | Höchstpreis pro angefangenem 24-h-Block (Tagesdeckel) |

**Berechnung** (stundenweise):
1. Dauer in Stunden, angefangene Stunden zählen voll (Minimum 1 Stunde).
2. `volleTage = stunden / 24`, `rest = stunden % 24`
3. `preis = volleTage × tagespreis + min(rest × stundenpreis, tagespreis)`

Beispiel Alpin (6 €/h, 30 €/Tag): 3 h = 18 €, 8 h = 30 €, 26 h = 30 € + 12 € = 42 €.

Der Tarif wird beim Buchen **in die Ausleihe kopiert** – spätere Preisänderungen wirken nur auf neue Buchungen.

### 1.4 Ausleihe / Reservierung – *b)*
| Feld | Bedeutung |
|---|---|
| `skiId`, `personId` | Wer leiht was |
| `beginn` | frei wählbarer Verleihbeginn (jetzt oder in der Zukunft = Reservierung) |
| `ende` | frei wählbare Verleihdauer (stundenweise bis mehrere Tage) |
| `preisCent` | vereinbarter Preis laut Tarif für `beginn`–`ende` |
| `rueckgabe` | Zeitpunkt der Rückgabe |
| `endpreisCent` | abgerechneter Preis nach tatsächlicher Dauer |

Regeln:
- `ende` muss nach `beginn` liegen; `beginn` darf höchstens 1 Stunde in der Vergangenheit liegen (Nachtragen einer gerade erfolgten Ausgabe).
- Ein Ski darf sich nicht in zwei offenen Ausleihen zeitlich überschneiden (Konflikt → HTTP 409).
- Ein überfälliger, nicht zurückgegebener Ski bleibt auf unbestimmte Zeit blockiert.
- **Status** (abgeleitet aus Zeit):
  - `reserviert` – Beginn liegt in der Zukunft
  - `aktiv` – zwischen Beginn und Ende
  - `überfällig` – Ende überschritten, noch nicht zurückgegeben
  - `abgeschlossen` – zurückgegeben
  - `storniert` – Reservierung storniert
- **Rückgabe** nur bei `aktiv`/`überfällig`. Abgerechnet wird stundenweise nach **tatsächlicher Dauer** (Beginn bis Rückgabe) mit dem gespeicherten Tarif – frühere Rückgabe wird billiger, Überziehung teurer.
- **Stornieren** nur bei `reserviert`.

### 1.5 Dashboard – *a)*
- Kennzahlen: verfügbare Ski (jetzt), ausgeliehene Ski, Gesamt Ski, registrierte Personen, überfällige Ausleihen, Reservierungen
- Schnellzugriff: Ski verwalten, Personen verwalten, Neue Ausleihe
- Liste der laufenden und kommenden Ausleihen mit Rückgabe-Knopf

## 2. Benutzeroberfläche
Navigation: **Dashboard · Ski · Personen · Ausleihen · Preise**

- **Neue Ausleihe** (Assistent in einem Formular):
  1. Person wählen (Suche) – Personen ohne Ausweis werden markiert, mit Link zum Nachtragen
  2. Beginn und Ende wählen; Schnellwahl *1 h, 2 h, 4 h, 1 Tag, 3 Tage, 1 Woche*
  3. Liste der im Zeitraum **freien** Ski mit berechnetem Preis, Filter nach Kategorie
  4. Buchen
- **Personen**: Anlegen/Bearbeiten inkl. Ausweistyp, Nummer und Datei-Upload; Ausweis ansehen
- **Preise**: Tarife je Kategorie bearbeiten, mit Live-Beispielrechnung

## 3. REST-API

| Methode | Pfad | Beschreibung |
|---|---|---|
| GET | `/api/dashboard` | Kennzahlen + laufende/kommende Ausleihen |
| GET, POST | `/api/ski` | Ski auflisten / anlegen |
| PUT, DELETE | `/api/ski/{id}` | Ski ändern / löschen |
| GET | `/api/ski/verfuegbar?von=&bis=` | freie Ski im Zeitraum inkl. Preis |
| GET, POST | `/api/personen` | Personen auflisten / anlegen |
| PUT, DELETE | `/api/personen/{id}` | Person ändern / löschen |
| POST | `/api/personen/{id}/ausweis` | Ausweis-Datei hochladen (multipart, Feld `datei`) |
| GET | `/api/personen/{id}/ausweis` | Ausweis-Datei ansehen |
| GET, POST | `/api/ausleihen` | Ausleihen auflisten / anlegen (`skiId, personId, beginn, ende`) |
| POST | `/api/ausleihen/{id}/rueckgabe` | Ski zurückgeben |
| POST | `/api/ausleihen/{id}/storno` | Reservierung stornieren |
| GET, PUT | `/api/preise` | Preisliste lesen / ändern |
| GET | `/api/preise/berechnen?kategorie=&von=&bis=` | Preisvorschau |

Zeitangaben im RFC-3339-Format (`2026-11-02T10:00:00+01:00`). Fehler: `{"fehler": "..."}` mit 400/404/409.

## 4. Projektstruktur
```
main.go                 Einstieg, Server, Seed
internal/domain/        Ski, Person, Ausweis, Tarif, Ausleihe (+ Tests)
internal/store/         thread-safer In-Memory-Store, Seed-Daten (+ Tests)
internal/api/           HTTP-Handler (+ Tests)
static/                 Frontend (HTML/CSS/JS)
```
