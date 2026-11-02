package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"skiverleih/internal/store"
)

var jetzt = time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)

func testServer() http.Handler {
	st := store.New()
	st.Seed(jetzt)
	mux := http.NewServeMux()
	New(st, func() time.Time { return jetzt }).Routes(mux)
	return mux
}

func anfrage(t *testing.T, h http.Handler, method, url string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, url, &buf))
	return rec
}

func TestDashboard(t *testing.T) {
	rec := anfrage(t, testServer(), "GET", "/api/dashboard", nil)
	var d store.Dashboard
	json.NewDecoder(rec.Body).Decode(&d)
	if rec.Code != 200 || d.GesamtSki != 20 || d.VerfuegbareSki != 17 {
		t.Errorf("%d %+v", rec.Code, d)
	}
}

func TestAusleiheStatuscodes(t *testing.T) {
	h := testServer()
	body := map[string]any{"skiId": 2, "personId": 21, "beginn": jetzt, "ende": jetzt.Add(3 * time.Hour)}
	if rec := anfrage(t, h, "POST", "/api/ausleihen", body); rec.Code != http.StatusCreated {
		t.Fatalf("anlegen: %d %s", rec.Code, rec.Body)
	}
	if rec := anfrage(t, h, "POST", "/api/ausleihen", body); rec.Code != http.StatusConflict {
		t.Errorf("doppelt: %d %s", rec.Code, rec.Body)
	}
	body["skiId"] = 999
	if rec := anfrage(t, h, "POST", "/api/ausleihen", body); rec.Code != http.StatusNotFound {
		t.Errorf("unbekannter Ski: %d", rec.Code)
	}
	if rec := anfrage(t, h, "GET", "/api/ski/verfuegbar?von=gestern", nil); rec.Code != http.StatusBadRequest {
		t.Errorf("schlechte Zeit: %d", rec.Code)
	}
}

func TestPreisBerechnen(t *testing.T) {
	url := "/api/preise/berechnen?kategorie=Alpin&von=2026-01-10T09:00:00Z&bis=2026-01-11T11:00:00Z"
	rec := anfrage(t, testServer(), "GET", url, nil)
	var res map[string]int
	json.NewDecoder(rec.Body).Decode(&res)
	if res["preisCent"] != 4200 || res["stunden"] != 26 {
		t.Errorf("%d %v", rec.Code, res)
	}
}

func TestAusweisUpload(t *testing.T) {
	h := testServer()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("datei", "ausweis.png")
	fw.Write([]byte("\x89PNG\r\n\x1a\n0000000000"))
	mw.Close()
	req := httptest.NewRequest("POST", "/api/personen/30/ausweis", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("Upload: %d %s", rec.Code, rec.Body)
	}
	rec = anfrage(t, h, "GET", "/api/personen/30/ausweis", nil)
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("Download: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
}
