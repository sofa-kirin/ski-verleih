// Ski-Verleih – Frontend. Spricht ausschließlich mit der Go-REST-API unter /api.
'use strict';

const app = document.getElementById('app');
const dialog = document.getElementById('dialog');
const dialogInhalt = document.getElementById('dialog-inhalt');

// ---------- Hilfsfunktionen ----------

async function api(methode, pfad, body) {
  const opt = { method: methode, headers: {} };
  if (body instanceof FormData) opt.body = body;
  else if (body !== undefined) {
    opt.body = JSON.stringify(body);
    opt.headers['Content-Type'] = 'application/json';
  }
  const res = await fetch('/api' + pfad, opt);
  if (res.status === 204) return null;
  const daten = await res.json();
  if (!res.ok) throw new Error(daten.fehler || res.statusText);
  return daten;
}

function esc(s) {
  return String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
}

const euro = cent => (cent / 100).toLocaleString('de-AT', { style: 'currency', currency: 'EUR' });
const datum = iso => new Date(iso).toLocaleString('de-AT', { weekday: 'short', day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });

// <input type="datetime-local"> arbeitet mit lokaler Zeit ohne Zeitzone.
function fuerInput(d) {
  const p = n => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}
const ausInput = s => new Date(s).toISOString();

function dauerText(von, bis) {
  const h = Math.ceil((bis - von) / 3600000);
  if (h < 24) return `${h} Std.`;
  const t = Math.floor(h / 24), r = h % 24;
  return `${t} ${t === 1 ? 'Tag' : 'Tage'}${r ? ` ${r} Std.` : ''}`;
}

let toastTimer;
function meldung(text, fehler = false) {
  const t = document.getElementById('toast');
  t.textContent = text;
  t.className = 'zeigen' + (fehler ? ' fehler' : '');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (t.className = ''), 3500);
}

// Führt eine Aktion aus und zeigt Fehler als Meldung an.
async function versuche(fn, erfolg) {
  try {
    const r = await fn();
    if (erfolg) meldung(erfolg);
    return r;
  } catch (e) {
    meldung(e.message, true);
    throw e;
  }
}

function oeffneDialog(titel, rumpf, fuss) {
  dialogInhalt.innerHTML = `
    <div class="dlg-kopf"><span>${esc(titel)}</span><button type="button" aria-label="Schließen" data-schliessen>&times;</button></div>
    <div class="dlg-rumpf">${rumpf}</div>
    <div class="dlg-fuss">${fuss}</div>`;
  dialogInhalt.querySelectorAll('[data-schliessen]').forEach(b => (b.onclick = () => dialog.close()));
  dialog.showModal();
  return dialogInhalt;
}

const statusBadge = s => `<span class="badge s-${esc(s)}">${esc(s)}</span>`;

// ---------- Router ----------

const seiten = { dashboard, ski: skiSeite, personen: personenSeite, ausleihen: ausleihenSeite, preise: preisSeite };

async function route() {
  const [name, param] = (location.hash.slice(1) || 'dashboard').split('/');
  const seite = seiten[name] || dashboard;
  document.querySelectorAll('#nav a').forEach(a => a.classList.toggle('aktiv', a.getAttribute('href') === '#' + (seiten[name] ? name : 'dashboard')));
  try {
    await seite(param);
  } catch (e) {
    app.innerHTML = `<div class="karte"><h1>Fehler</h1><p>${esc(e.message)}</p></div>`;
  }
}
window.addEventListener('hashchange', route);
const neuLaden = () => route();

// ---------- Dashboard ----------

async function dashboard() {
  const d = await api('GET', '/dashboard');
  const heute = new Date().toLocaleDateString('de-AT', { weekday: 'long', day: 'numeric', month: 'long' });
  const band = 'Vertrauen // Sicherheit // Stundenweise // Pulverschnee // Flexibel // Piste frei // ';
  app.innerHTML = `
    <section class="hero">
      <div class="hero-innen">
        <span class="sticker">Dashboard · ${heute}</span>
        <h1 class="hero-titel">Ski<span class="outline">verleih</span></h1>
        <p class="hero-text">${d.verfuegbareSki} Paar Ski sind bereit für die Piste.
          ${d.ueberfaellig ? `<strong>${d.ueberfaellig} Rückgabe(n) überfällig.</strong>` : ''}</p>
        <button class="btn weiss gross kringel" id="neu">⛷ Jetzt ausleihen</button>
      </div>
      <svg class="hero-berge" viewBox="0 0 1440 260" preserveAspectRatio="none" aria-hidden="true">
        <path class="berg-1" d="M0 260V150l120-70 70 40 150-110 90 60 70-30 160 120 110-80 140 90 90-50 150 100 110-70 180 120v110Z"/>
        <path class="berg-2" d="M0 260V215l60-14 40 10 70-22 50 16 90-28 60 20 80-16 70 24 110-30 80 22 60-12 120 26 90-18 100 22 80-14 100 20 70-10 80 12 60-8v87Z"/>
      </svg>
    </section>

    <div class="band" aria-hidden="true"><div>${band.repeat(4)}</div></div>

    <div class="kennzahlen">
      <a class="kennzahl" href="#ski"><span class="nr">/1</span><div class="wert">${d.verfuegbareSki}</div><div class="titel">Verfügbare Ski</div></a>
      <a class="kennzahl blau" href="#ausleihen"><span class="nr">/2</span><div class="wert">${d.ausgelieheneSki}</div><div class="titel">Ausgeliehene Ski</div></a>
      <a class="kennzahl" href="#ski"><span class="nr">/3</span><div class="wert">${d.gesamtSki}</div><div class="titel">Gesamt Ski</div></a>
      <a class="kennzahl" href="#personen"><span class="nr">/4</span><div class="wert">${d.personen}</div><div class="titel">Registrierte Personen</div></a>
    </div>

    <div class="karte">
      <div class="karte-kopf">
        <h2>Auf der Piste</h2>
        <div class="hinweise">
          ${d.ueberfaellig ? `<span class="hinweis warn">${d.ueberfaellig} überfällig</span>` : ''}
          <span class="hinweis info">${d.reservierungen} Reservierung(en)</span>
        </div>
      </div>
      ${ausleihTabelle(d.offen)}
    </div>

    <section class="schnellzugriff">
      <h2>Schnell&shy;zugriff</h2>
      <div class="schnell">
        <a class="btn" href="#ski">Ski verwalten</a>
        <a class="btn" href="#personen">Personen verwalten</a>
        <a class="btn" href="#preise">Preise anpassen</a>
      </div>
    </section>`;
  document.getElementById('neu').onclick = () => neueAusleihe();
  aktionenBinden();
}

function ausleihTabelle(liste) {
  if (!liste.length) return '<p class="leer">Keine Ausleihen vorhanden.</p>';
  return `<div class="tabelle-scroll"><table>
    <thead><tr><th>Status</th><th>Ski</th><th>Person</th><th>Beginn</th><th>Ende</th><th>Preis</th><th></th></tr></thead>
    <tbody>${liste.map(a => `
      <tr>
        <td>${statusBadge(a.status)}</td>
        <td>${esc(a.ski)}</td>
        <td>${esc(a.person)}</td>
        <td>${datum(a.beginn)}</td>
        <td>${datum(a.ende)}<br><small>${dauerText(new Date(a.beginn), new Date(a.ende))}</small></td>
        <td>${a.rueckgabe ? `<strong>${euro(a.endpreisCent)}</strong><br><small>vereinbart ${euro(a.preisCent)}</small>` : euro(a.preisCent)}</td>
        <td class="aktionen">
          ${a.status === 'aktiv' || a.status === 'überfällig' ? `<button class="btn klein gruen" data-rueckgabe="${a.id}">Rückgabe</button>` : ''}
          ${a.status === 'reserviert' ? `<button class="btn klein rot" data-storno="${a.id}">Stornieren</button>` : ''}
        </td>
      </tr>`).join('')}</tbody></table></div>`;
}

function aktionenBinden() {
  app.querySelectorAll('[data-rueckgabe]').forEach(b => (b.onclick = async () => {
    const a = await versuche(() => api('POST', `/ausleihen/${b.dataset.rueckgabe}/rueckgabe`));
    meldung(`Zurückgegeben – zu zahlen: ${euro(a.endpreisCent)}`);
    neuLaden();
  }));
  app.querySelectorAll('[data-storno]').forEach(b => (b.onclick = async () => {
    if (!confirm('Reservierung wirklich stornieren?')) return;
    await versuche(() => api('POST', `/ausleihen/${b.dataset.storno}/storno`), 'Reservierung storniert');
    neuLaden();
  }));
}

// ---------- Neue Ausleihe ----------

async function neueAusleihe(vorgewaehltePerson) {
  const personen = await api('GET', '/personen');
  const jetzt = new Date();
  jetzt.setSeconds(0, 0);
  const zustand = { personId: vorgewaehltePerson || '', von: jetzt, bis: new Date(+jetzt + 4 * 3600000), skiId: null, kategorie: '', angebote: [] };

  const d = oeffneDialog('Neue Ausleihe', `
    <div class="schritt">1. Person</div>
    <select id="na-person">
      <option value="">– Person wählen –</option>
      ${personen.map(p => `<option value="${p.id}">${esc(p.nachname)}, ${esc(p.vorname)}${hatAusweis(p) ? '' : ' (kein Ausweis!)'}</option>`).join('')}
    </select>
    <div id="na-ausweis"></div>

    <div class="schritt">2. Zeitraum</div>
    <div class="formular">
      <label>Beginn <input type="datetime-local" id="na-von"></label>
      <label>Ende <input type="datetime-local" id="na-bis"></label>
    </div>
    <div class="dauer-knoepfe">
      ${[[1, '1 Std.'], [2, '2 Std.'], [4, '4 Std.'], [24, '1 Tag'], [72, '3 Tage'], [168, '1 Woche']]
        .map(([h, t]) => `<button type="button" class="chip" data-stunden="${h}">${t}</button>`).join('')}
    </div>

    <div class="schritt">3. Freie Ski in diesem Zeitraum</div>
    <div class="filter" id="na-kat"></div>
    <div class="angebote" id="na-angebote"></div>`,
    `<span class="zusammenfassung" id="na-summe">Bitte Person, Zeitraum und Ski wählen</span>
     <button class="btn grau" data-schliessen>Abbrechen</button>
     <button class="btn gruen" id="na-buchen" disabled>Buchen</button>`);

  const $ = s => d.querySelector(s);
  $('#na-person').value = zustand.personId;
  $('#na-von').value = fuerInput(zustand.von);
  $('#na-bis').value = fuerInput(zustand.bis);

  function personPruefen() {
    const p = personen.find(p => p.id == zustand.personId);
    $('#na-ausweis').innerHTML = p && !hatAusweis(p)
      ? `<div class="warnung">Für ${esc(p.vorname)} ist kein Ausweis hinterlegt. <a href="#personen" id="na-ausweis-link">Ausweis jetzt erfassen</a></div>` : '';
    const link = $('#na-ausweis-link');
    if (link) link.onclick = e => { e.preventDefault(); dialog.close(); personDialog(p); };
    aktualisieren();
  }

  async function angeboteLaden() {
    zustand.von = new Date($('#na-von').value);
    zustand.bis = new Date($('#na-bis').value);
    const box = $('#na-angebote');
    if (!(zustand.bis > zustand.von)) {
      box.innerHTML = '<p class="warnung">Das Ende muss nach dem Beginn liegen.</p>';
      zustand.angebote = [];
      return aktualisieren();
    }
    const q = new URLSearchParams({ von: zustand.von.toISOString(), bis: zustand.bis.toISOString() });
    zustand.angebote = await api('GET', '/ski/verfuegbar?' + q);
    if (!zustand.angebote.some(a => a.id === zustand.skiId)) zustand.skiId = null;
    angeboteZeigen();
  }

  function angeboteZeigen() {
    const kats = [...new Set(zustand.angebote.map(a => a.kategorie))];
    $('#na-kat').innerHTML = ['', ...kats].map(k =>
      `<button type="button" class="chip ${k === zustand.kategorie ? 'aktiv' : ''}" data-kat="${esc(k)}">${k ? esc(k) : 'Alle'}</button>`).join('');
    $('#na-kat').querySelectorAll('[data-kat]').forEach(b => (b.onclick = () => { zustand.kategorie = b.dataset.kat; angeboteZeigen(); }));
    const liste = zustand.angebote.filter(a => !zustand.kategorie || a.kategorie === zustand.kategorie);
    $('#na-angebote').innerHTML = liste.length ? liste.map(a => `
      <div class="angebot ${a.id === zustand.skiId ? 'gewaehlt' : ''}" data-ski="${a.id}" tabindex="0">
        <span class="kat">${esc(a.kategorie)} · ${a.laengeCm} cm</span>
        <span>${esc(a.marke)} ${esc(a.modell)}</span>
        <span class="preis">${euro(a.preisCent)}</span>
      </div>`).join('') : '<p class="leer">In diesem Zeitraum ist kein Ski frei.</p>';
    $('#na-angebote').querySelectorAll('[data-ski]').forEach(el => {
      el.onclick = el.onkeydown = e => {
        if (e.type === 'keydown' && e.key !== 'Enter' && e.key !== ' ') return;
        zustand.skiId = +el.dataset.ski;
        angeboteZeigen();
      };
    });
    aktualisieren();
  }

  function aktualisieren() {
    const angebot = zustand.angebote.find(a => a.id === zustand.skiId);
    const p = personen.find(p => p.id == zustand.personId);
    const ok = angebot && p && hatAusweis(p);
    $('#na-buchen').disabled = !ok;
    $('#na-summe').innerHTML = angebot
      ? `${esc(angebot.marke)} ${esc(angebot.modell)} · ${dauerText(zustand.von, zustand.bis)} · <strong>${euro(angebot.preisCent)}</strong>`
      : 'Bitte Person, Zeitraum und Ski wählen';
  }

  $('#na-person').onchange = e => { zustand.personId = e.target.value; personPruefen(); };
  $('#na-von').onchange = () => {
    // Dauer beibehalten, wenn der Beginn verschoben wird
    const dauer = zustand.bis - zustand.von;
    $('#na-bis').value = fuerInput(new Date(+new Date($('#na-von').value) + dauer));
    angeboteLaden();
  };
  $('#na-bis').onchange = angeboteLaden;
  d.querySelectorAll('[data-stunden]').forEach(b => (b.onclick = () => {
    $('#na-bis').value = fuerInput(new Date(+new Date($('#na-von').value) + b.dataset.stunden * 3600000));
    angeboteLaden();
  }));
  $('#na-buchen').onclick = async () => {
    const a = await versuche(() => api('POST', '/ausleihen', {
      skiId: zustand.skiId, personId: +zustand.personId,
      beginn: ausInput($('#na-von').value), ende: ausInput($('#na-bis').value),
    }));
    dialog.close();
    meldung(`${a.status === 'reserviert' ? 'Reserviert' : 'Ausgeliehen'}: ${a.ski} an ${a.person} (${euro(a.preisCent)})`);
    neuLaden();
  };

  personPruefen();
  await angeboteLaden();
}

const hatAusweis = p => (p.ausweis.typ && p.ausweis.nummer) || p.ausweis.dateiname;

// ---------- Ski-Verwaltung ----------

async function skiSeite() {
  const [ski, ausleihen] = await Promise.all([api('GET', '/ski'), api('GET', '/ausleihen')]);
  const unterwegs = new Set(ausleihen.filter(a => a.status === 'aktiv' || a.status === 'überfällig').map(a => a.skiId));
  app.innerHTML = `
    <div class="karte">
      <h1>Ski-Verwaltung <button class="btn gruen" id="neu">+ Ski hinzufügen</button></h1>
      <div class="filter"><input class="suche" id="suche" placeholder="Suchen (Marke, Modell, Kategorie) …"></div>
      <div class="tabelle-scroll"><table>
        <thead><tr><th>Nr.</th><th>Marke / Modell</th><th>Kategorie</th><th>Länge</th><th>Zustand</th><th></th></tr></thead>
        <tbody id="zeilen"></tbody>
      </table></div>
    </div>`;
  const zeigen = filter => {
    const f = filter.toLowerCase();
    const liste = ski.filter(s => `${s.marke} ${s.modell} ${s.kategorie}`.toLowerCase().includes(f));
    document.getElementById('zeilen').innerHTML = liste.map(s => `
      <tr>
        <td>#${s.id}</td>
        <td><strong>${esc(s.marke)}</strong> ${esc(s.modell)}</td>
        <td>${esc(s.kategorie)}</td>
        <td>${s.laengeCm} cm</td>
        <td>${unterwegs.has(s.id) ? statusBadge('aktiv').replace('>aktiv<', '>ausgeliehen<') : '<span class="badge s-ok">im Lager</span>'}</td>
        <td class="aktionen">
          <button class="btn klein" data-bearbeiten="${s.id}">Bearbeiten</button>
          <button class="btn klein rot" data-loeschen="${s.id}">Löschen</button>
        </td>
      </tr>`).join('') || '<tr><td colspan="6" class="leer">Keine Ski gefunden.</td></tr>';
    app.querySelectorAll('[data-bearbeiten]').forEach(b => (b.onclick = () => skiDialog(ski.find(s => s.id == b.dataset.bearbeiten))));
    app.querySelectorAll('[data-loeschen]').forEach(b => (b.onclick = async () => {
      if (!confirm('Ski wirklich löschen?')) return;
      await versuche(() => api('DELETE', '/ski/' + b.dataset.loeschen), 'Ski gelöscht');
      neuLaden();
    }));
  };
  document.getElementById('suche').oninput = e => zeigen(e.target.value);
  document.getElementById('neu').onclick = () => skiDialog();
  zeigen('');
}

async function skiDialog(ski = { marke: '', modell: '', kategorie: '', laengeCm: 170 }) {
  const kategorien = Object.keys(await api('GET', '/preise')).sort();
  const d = oeffneDialog(ski.id ? `Ski #${ski.id} bearbeiten` : 'Ski hinzufügen', `
    <form class="formular" id="f">
      <label>Marke <input name="marke" required value="${esc(ski.marke)}"></label>
      <label>Modell <input name="modell" required value="${esc(ski.modell)}"></label>
      <label>Kategorie <select name="kategorie" required>
        ${kategorien.map(k => `<option ${k === ski.kategorie ? 'selected' : ''}>${esc(k)}</option>`).join('')}
      </select></label>
      <label>Länge (cm) <input name="laengeCm" type="number" min="70" max="220" required value="${ski.laengeCm}"></label>
    </form>`,
    `<button class="btn grau" data-schliessen>Abbrechen</button><button class="btn gruen" id="speichern">Speichern</button>`);
  d.querySelector('#speichern').onclick = async () => {
    const f = d.querySelector('#f');
    if (!f.reportValidity()) return;
    const daten = Object.fromEntries(new FormData(f));
    daten.laengeCm = +daten.laengeCm;
    await versuche(() => ski.id ? api('PUT', '/ski/' + ski.id, daten) : api('POST', '/ski', daten), 'Ski gespeichert');
    dialog.close();
    neuLaden();
  };
}

// ---------- Personen ----------

async function personenSeite() {
  const personen = await api('GET', '/personen');
  app.innerHTML = `
    <div class="karte">
      <h1>Personen <button class="btn gruen" id="neu">+ Person registrieren</button></h1>
      <div class="filter"><input class="suche" id="suche" placeholder="Name, E-Mail oder Ausweisnummer …"></div>
      <div class="tabelle-scroll"><table>
        <thead><tr><th>Name</th><th>Kontakt</th><th>Ausweis</th><th></th></tr></thead>
        <tbody id="zeilen"></tbody>
      </table></div>
    </div>`;
  const zeigen = filter => {
    const f = filter.toLowerCase();
    const liste = personen.filter(p => `${p.vorname} ${p.nachname} ${p.email} ${p.ausweis.nummer}`.toLowerCase().includes(f));
    document.getElementById('zeilen').innerHTML = liste.map(p => `
      <tr>
        <td><strong>${esc(p.nachname)}</strong>, ${esc(p.vorname)}</td>
        <td>${esc(p.email)}${p.email && p.telefon ? '<br>' : ''}${esc(p.telefon)}</td>
        <td>
          ${p.ausweis.typ ? `${esc(p.ausweis.typ)}<br><small>Nr. ${esc(p.ausweis.nummer)}</small>` : ''}
          ${p.ausweis.dateiname ? `<br><a href="/api/personen/${p.id}/ausweis" target="_blank">📎 ${esc(p.ausweis.dateiname)}</a>` : ''}
          ${hatAusweis(p) ? '' : '<span class="badge s-fehlt">fehlt</span>'}
        </td>
        <td class="aktionen">
          <button class="btn klein gruen" data-leihen="${p.id}">Ausleihen</button>
          <button class="btn klein" data-bearbeiten="${p.id}">Bearbeiten</button>
          <button class="btn klein rot" data-loeschen="${p.id}">Löschen</button>
        </td>
      </tr>`).join('') || '<tr><td colspan="4" class="leer">Keine Personen gefunden.</td></tr>';
    app.querySelectorAll('[data-leihen]').forEach(b => (b.onclick = () => neueAusleihe(b.dataset.leihen)));
    app.querySelectorAll('[data-bearbeiten]').forEach(b => (b.onclick = () => personDialog(personen.find(p => p.id == b.dataset.bearbeiten))));
    app.querySelectorAll('[data-loeschen]').forEach(b => (b.onclick = async () => {
      if (!confirm('Person wirklich löschen?')) return;
      await versuche(() => api('DELETE', '/personen/' + b.dataset.loeschen), 'Person gelöscht');
      neuLaden();
    }));
  };
  document.getElementById('suche').oninput = e => zeigen(e.target.value);
  document.getElementById('neu').onclick = () => personDialog();
  zeigen('');
}

async function personDialog(p = { vorname: '', nachname: '', email: '', telefon: '', ausweis: { typ: '', nummer: '' } }) {
  const typen = await api('GET', '/ausweistypen');
  const d = oeffneDialog(p.id ? 'Person bearbeiten' : 'Person registrieren', `
    <form class="formular" id="f">
      <label>Vorname <input name="vorname" required value="${esc(p.vorname)}" autocomplete="off"></label>
      <label>Nachname <input name="nachname" required value="${esc(p.nachname)}" autocomplete="off"></label>
      <label>E-Mail <small>optional</small><input name="email" type="email" value="${esc(p.email)}"></label>
      <label>Telefon <small>optional</small><input name="telefon" type="tel" value="${esc(p.telefon)}"></label>
      <fieldset class="breit">
        <legend>Ausweis</legend>
        <label>Typ <select name="typ">
          <option value="">– bitte wählen –</option>
          ${typen.map(t => `<option ${t === p.ausweis.typ ? 'selected' : ''}>${esc(t)}</option>`).join('')}
        </select></label>
        <label>Ausweisnummer <input name="nummer" value="${esc(p.ausweis.nummer)}" autocomplete="off"></label>
        <div class="dropzone" id="drop">
          ${p.ausweis.dateiname ? `Aktuell: <a href="/api/personen/${p.id}/ausweis" target="_blank">${esc(p.ausweis.dateiname)}</a><br>` : ''}
          <span id="drop-text">📎 Foto oder Scan hierher ziehen oder <u>klicken</u> (JPG, PNG, PDF, max. 5 MB)</span>
          <input type="file" id="datei" accept="image/jpeg,image/png,application/pdf" hidden>
        </div>
      </fieldset>
      <p class="breit beispiele">Für eine Ausleihe muss ein Ausweis hinterlegt sein – Typ + Nummer eingeben und/oder eine Datei hochladen.</p>
    </form>`,
    `<button class="btn grau" data-schliessen>Abbrechen</button><button class="btn gruen" id="speichern">Speichern</button>`);

  let datei = null;
  const drop = d.querySelector('#drop'), input = d.querySelector('#datei');
  const waehle = f => {
    if (!f) return;
    if (f.size > 5 * 1024 * 1024) return meldung('Die Datei ist größer als 5 MB', true);
    datei = f;
    d.querySelector('#drop-text').textContent = `✔ ${f.name} wird beim Speichern hochgeladen`;
  };
  drop.onclick = e => { if (e.target.tagName !== 'A') input.click(); };
  input.onchange = () => waehle(input.files[0]);
  drop.ondragover = e => { e.preventDefault(); drop.classList.add('drueber'); };
  drop.ondragleave = () => drop.classList.remove('drueber');
  drop.ondrop = e => { e.preventDefault(); drop.classList.remove('drueber'); waehle(e.dataTransfer.files[0]); };

  d.querySelector('#speichern').onclick = async () => {
    const f = d.querySelector('#f');
    if (!f.reportValidity()) return;
    const w = Object.fromEntries(new FormData(f));
    const daten = { vorname: w.vorname, nachname: w.nachname, email: w.email, telefon: w.telefon, ausweis: { typ: w.typ, nummer: w.nummer } };
    const gespeichert = await versuche(() => p.id ? api('PUT', '/personen/' + p.id, daten) : api('POST', '/personen', daten));
    if (datei) {
      const fd = new FormData();
      fd.append('datei', datei);
      await versuche(() => api('POST', `/personen/${gespeichert.id}/ausweis`, fd));
    }
    dialog.close();
    meldung('Person gespeichert');
    neuLaden();
  };
}

// ---------- Ausleihen ----------

let ausleihFilter = 'offen';

async function ausleihenSeite() {
  const alle = await api('GET', '/ausleihen');
  const filter = {
    offen: ['Offen', a => ['aktiv', 'überfällig', 'reserviert'].includes(a.status)],
    aktiv: ['Ausgeliehen', a => a.status === 'aktiv' || a.status === 'überfällig'],
    reserviert: ['Reserviert', a => a.status === 'reserviert'],
    abgeschlossen: ['Abgeschlossen', a => a.status === 'abgeschlossen' || a.status === 'storniert'],
    alle: ['Alle', () => true],
  };
  app.innerHTML = `
    <div class="karte">
      <h1>Ausleihen &amp; Reservierungen <button class="btn gruen" id="neu">Neue Ausleihe</button></h1>
      <div class="filter">${Object.entries(filter).map(([k, [t, f]]) =>
        `<button class="chip ${k === ausleihFilter ? 'aktiv' : ''}" data-f="${k}">${t} (${alle.filter(f).length})</button>`).join('')}</div>
      ${ausleihTabelle(alle.filter(filter[ausleihFilter][1]))}
    </div>`;
  app.querySelectorAll('[data-f]').forEach(b => (b.onclick = () => { ausleihFilter = b.dataset.f; neuLaden(); }));
  document.getElementById('neu').onclick = () => neueAusleihe();
  aktionenBinden();
}

// ---------- Preise ----------

async function preisSeite() {
  const preise = await api('GET', '/preise');
  const zeilen = Object.entries(preise).sort(([a], [b]) => a.localeCompare(b));
  app.innerHTML = `
    <div class="karte">
      <h1>Preise</h1>
      <p>Abgerechnet wird <strong>stundenweise</strong> (jede angefangene Stunde). Pro angefangenem Tag (24 h) wird höchstens der <strong>Tagespreis</strong> berechnet.
         Änderungen gelten für neue Buchungen; bestehende Ausleihen behalten ihren Tarif.</p>
      <div class="tabelle-scroll"><table id="tarife">
        <thead><tr><th>Kategorie</th><th>pro Stunde (€)</th><th>pro Tag (€)</th><th>Beispiele</th><th></th></tr></thead>
        <tbody>${zeilen.map(([k, t]) => tarifZeile(k, t)).join('')}</tbody>
      </table></div>
      <div class="schnell" style="margin-top:1rem">
        <button class="btn grau" id="neue-kat">+ Kategorie</button>
        <button class="btn gruen" id="speichern">Preise speichern</button>
      </div>
    </div>`;
  const tbody = app.querySelector('#tarife tbody');
  const binden = () => {
    tbody.querySelectorAll('tr').forEach(tr => {
      tr.querySelectorAll('input').forEach(i => (i.oninput = () => beispiele(tr)));
      tr.querySelector('[data-weg]').onclick = () => tr.remove();
      beispiele(tr);
    });
  };
  document.getElementById('neue-kat').onclick = () => {
    tbody.insertAdjacentHTML('beforeend', tarifZeile('', { stundenpreisCent: 500, tagespreisCent: 2500 }, true));
    binden();
  };
  document.getElementById('speichern').onclick = async () => {
    const liste = {};
    for (const tr of tbody.querySelectorAll('tr')) {
      const kat = tr.querySelector('[name=kat]').value.trim();
      if (!kat) return meldung('Jede Kategorie braucht einen Namen', true);
      liste[kat] = { stundenpreisCent: centAus(tr, 'std'), tagespreisCent: centAus(tr, 'tag') };
    }
    await versuche(() => api('PUT', '/preise', liste), 'Preise gespeichert');
    neuLaden();
  };
  binden();
}

function tarifZeile(kat, t, neu = false) {
  return `<tr class="tarif-zeile">
    <td><input name="kat" value="${esc(kat)}" ${neu ? '' : 'readonly'} placeholder="z.B. Snowboard"></td>
    <td><input name="std" type="number" min="0.5" step="0.5" value="${t.stundenpreisCent / 100}"></td>
    <td><input name="tag" type="number" min="0.5" step="0.5" value="${t.tagespreisCent / 100}"></td>
    <td class="beispiele"></td>
    <td class="aktionen"><button class="btn klein rot" data-weg title="Kategorie entfernen">✕</button></td>
  </tr>`;
}

const centAus = (tr, name) => Math.round(parseFloat(tr.querySelector(`[name=${name}]`).value || '0') * 100);

// Gleiche Formel wie domain.Tarif.Berechne – nur zur Vorschau.
function preisVorschau(std, tag, stunden) {
  return Math.floor(stunden / 24) * tag + Math.min((stunden % 24) * std, tag);
}

function beispiele(tr) {
  const std = centAus(tr, 'std'), tag = centAus(tr, 'tag');
  tr.querySelector('.beispiele').textContent = tag < std
    ? '⚠ Tagespreis kleiner als Stundenpreis'
    : [[2, '2 h'], [8, '8 h'], [26, '26 h'], [72, '3 Tage']].map(([h, t]) => `${t}: ${euro(preisVorschau(std, tag, h))}`).join(' · ');
}

route();
