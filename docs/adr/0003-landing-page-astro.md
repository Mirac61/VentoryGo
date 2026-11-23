# 3. Landing Page: Astro

Status: Vorgeschlagen
Datum: 2026-11-03
Ticket: [#103](https://github.com/Mirac61/VentoryGo/issues/103)

## Entscheidung

Astro, mit React-Islands nur dort, wo etwas interaktiv ist. Die Seite bleibt
statisches HTML (SEO, Ladezeit), bekommt aber Komponenten und einen Build-Schritt.

## Verworfene Alternativen

**Plain HTML/CSS/JS.** ...
**Vite + React als SPA.** Leeres `<div id="root">` für Crawler, schlechterer First Paint.
**Next.js.** Keine Server-Features nötig, schwereres Toolchain für statische Ausgabe.

## Konsequenzen

- Dev-Server auf :4173, separat über `pnpm dev` in `landing/`
- `mprocs.yaml` und die Dev-Skripte starten Backend, Frontend und Dev-Proxy; die
  Landing Page ist nicht Teil des lokalen Stacks und wird separat gestartet
- `shared/fonts` per `vite.server.fs.allow` erreichbar
- `landing/` nutzt pnpm, `frontend/` npm (bis zur Migration)
- Produktion: gehostete Instanz liefert Landing Page und App hinter demselben
  Reverse-Proxy, Self-Hosted startet beim Login

## Offene Punkte

- pnpm-Migration für `frontend/`