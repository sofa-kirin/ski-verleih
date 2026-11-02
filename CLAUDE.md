# Ski-Verleih (Prototyp)

Nur Prototypen. Priorität: schnell verständlich, leicht änderbar.
Die fachliche Spezifikation steht in `README.md` – sie ist die Quelle der Wahrheit (Spec-Driven Development).
Vor jeder Änderung am Code zuerst die Spezifikation anpassen, dann implementieren, dann testen.

## Stack
- Backend: Go ('objektorientiert': Structs + Methoden, Interfaces wo sinnvoll)
- DB: In-Memory (kein Postgres/SQLite, kein Object-Relational Mapping (ORM) nötig)
- Frontend: reines HTML/JS (eine Single-Page-App ohne Build-Schritt, per `embed` im Binary)
- Versionierung: lokales Git, bei Bedarf GitHub

## Commands
- Backend starten: `go run .` → http://localhost:8080 (Port per `-addr :9000` änderbar)
- Tests: `go test ./...`
- Format: `gofmt -w .` bzw. `go fmt ./...`
- Vet: `go vet ./...`
- Frontend: wird vom Go-Server unter `/` ausgeliefert, kein eigener Server nötig

## Architektur (Go)
- `main.go` → Einstieg, HTTP-Server, Seed-Daten, Auslieferung von `static/`
- `internal/domain/` → Kern-Structs + Methoden (Business-Logik: Preise, Status, Überschneidung, Validierung)
- `internal/store/` → In-Memory-Store (Maps + `sync.RWMutex`), fachliche Prüfungen über mehrere Entitäten
- `internal/api/` → HTTP-Handler (JSON), Mapping von Fehlern auf Statuscodes
- `static/` → `index.html`, `app.js`, `style.css`

## Fehlerbehandlung
- Validierungsfehler als `store.ErrUngueltig`, fehlende Objekte als `store.ErrNichtGefunden`, Belegungskonflikte als `store.ErrKonflikt`
- API mappt diese auf 400 / 404 / 409 und liefert `{"fehler": "..."}`
- Fehler mit `fmt.Errorf("%w", err)` wrappen, nicht verschlucken

## Konventionen
- Methoden an Structs hängen statt freier Funktionen, wo es Sinn ergibt
- Interfaces nur bei echtem Bedarf (z.B. Store-Interface für spätere Austauschbarkeit)
- Kein Dependency Injection Framework
- Keine neuen Dependencies ohne Nachfrage (nur Standardbibliothek)
- In-Memory-Store muss thread-safe sein
- Keine Secrets im Code (auch bei Prototypen nicht)
- Geldbeträge immer als Integer in Cent (`...Cent`), nie als float
- Zeitpunkte als `time.Time`, im JSON RFC 3339; die aktuelle Zeit wird injiziert (`now`), damit Tests deterministisch sind
- Bezeichner und UI-Texte auf Deutsch

## Frontend
- HTML-Variante: möglichst minimales JS, klare Trennung von Backend-API
- API-Calls immer gegen die Go-Endpoints, nie direkt gegen In-Memory-Strukturen
- Benutzerfreundlich: Formulare mit sinnvollen Vorbelegungen, Fehlermeldungen als Hinweis, Preisvorschau vor dem Buchen

## Regeln
- Keine generierten Dateien von Hand editieren
- Commit-Messages: kurz und imperativ
- Bei Unklarheit fragen statt raten
- Prototyp-Fokus: lieber einfach und klar als elegant und komplex
