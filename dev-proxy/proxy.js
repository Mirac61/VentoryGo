// Lokaler Reverse-Proxy für die Entwicklung: führt Landing Page (statischer
// Server, z.B. `npx serve .` im REPO-ROOT, nicht in landing/) und Frontend
// (`npm run dev` auf :5173) unter einem gemeinsamen Origin zusammen, damit
// relative Links wie href="/login" in landing/index.html lokal genauso
// funktionieren wie später hinter einem echten Reverse-Proxy in Produktion.
//
// WICHTIG: `serve` muss im Repo-Root laufen (nicht in landing/), sonst kann
// die Landing Page nicht auf den Geschwisterordner shared/fonts/ zugreifen
// (serve serviert nur den angegebenen Ordner und alles darunter, nichts
// darüber — dasselbe Prinzip wie Vites fs.allow-Einschränkung).
//
// Nutzung:
//   node proxy.js
//
// Keine Abhängigkeiten nötig — nutzt ausschließlich Node's eingebautes
// http-Modul. (Das ursprüngliche http-proxy-Paket ist mit aktuellen
// Node-Versionen inkompatibel, siehe TypeError beim Laden.)
//
// Routing:
//   /login, /register, /dashboard, /api/*  → :5173 (Frontend/Vite)
//   /                                       → :4173/landing/index.html
//   alles andere (shared/, landing/*)       → :4173 (Landing/Assets, unverändert)

const http = require('node:http')

const PROXY_PORT = 8090
const FRONTEND_HOST = 'localhost'
const FRONTEND_PORT = 5173
const LANDING_HOST = 'localhost'
const LANDING_PORT = 4173

// Pfade, die zur React-App gehören. Alles, was hier nicht matcht, geht an
// die Landing Page. Bei neuen Routen (z.B. /dashboard/settings) ggf. erweitern.
const FRONTEND_PREFIXES = ['/login', '/register', '/dashboard', '/api', '/@vite', '/@react-refresh', '/@fs', '/src', '/node_modules']

function isFrontendPath(url) {
    return FRONTEND_PREFIXES.some((prefix) => url.startsWith(prefix))
}

function targetFor(url) {
    return isFrontendPath(url)
        ? { host: FRONTEND_HOST, port: FRONTEND_PORT }
        : { host: LANDING_HOST, port: LANDING_PORT }
}

// `serve` läuft jetzt im Repo-Root, die Landing Page liegt unter
// landing/index.html. Sowohl "/" als auch "/landing" und "/landing/" werden
// serverseitig direkt auf /landing/index.html aufgelöst — bewusst KEIN
// Redirect an den Browser, das hatte mit serves eigenem Verzeichnis-Handling
// zu einer Redirect-Schleife geführt. Relative Assets in index.html
// (styles.css, images/...) funktionieren trotzdem korrekt, weil der Browser
// die tatsächlich aufgerufene URL (/landing oder /landing/) als Basis nimmt,
// nicht den intern umgeschriebenen Pfad.
function resolveLandingPath(url) {
    if (url === '/' || url === '/landing' || url === '/landing/') {
        return '/landing/index.html'
    }
    return url
}

function proxyRequest(clientReq, clientRes) {
    if (isFrontendPath(clientReq.url)) {
        return proxyTo(clientReq, clientRes, { host: FRONTEND_HOST, port: FRONTEND_PORT }, clientReq.url)
    }

    const path = resolveLandingPath(clientReq.url)
    proxyTo(clientReq, clientRes, { host: LANDING_HOST, port: LANDING_PORT }, path)
}

function proxyTo(clientReq, clientRes, target, path) {
    const upstreamReq = http.request(
        {
            host: target.host,
            port: target.port,
            path,
            method: clientReq.method,
            headers: clientReq.headers,
        },
        (upstreamRes) => {
            clientRes.writeHead(upstreamRes.statusCode ?? 502, upstreamRes.headers)
            upstreamRes.pipe(clientRes)
        },
    )

    upstreamReq.on('error', (err) => {
        console.error(`Proxy-Fehler (${target.host}:${target.port}):`, err.message)
        if (!clientRes.headersSent) {
            clientRes.writeHead(502, { 'Content-Type': 'text/plain' })
        }
        clientRes.end(`Proxy-Fehler — läuft der Zielserver auf Port ${target.port}?`)
    })

    clientReq.pipe(upstreamReq)
}

const server = http.createServer(proxyRequest)

// Vite braucht WebSocket für Hot Module Replacement — ohne das hier bricht HMR.
server.on('upgrade', (req, clientSocket, head) => {
    const target = targetFor(req.url)

    const upstreamReq = http.request({
        host: target.host,
        port: target.port,
        path: req.url,
        method: req.method,
        headers: req.headers,
    })

    upstreamReq.on('upgrade', (upstreamRes, upstreamSocket, upstreamHead) => {
        clientSocket.write(
            `HTTP/1.1 101 Switching Protocols\r\n` +
            Object.entries(upstreamRes.headers)
                .map(([key, value]) => `${key}: ${value}`)
                .join('\r\n') +
            '\r\n\r\n',
        )
        upstreamSocket.write(upstreamHead)
        upstreamSocket.pipe(clientSocket)
        clientSocket.pipe(upstreamSocket)
    })

    upstreamReq.on('error', (err) => {
        console.error(`WebSocket-Proxy-Fehler (${target.host}:${target.port}):`, err.message)
        clientSocket.destroy()
    })

    upstreamReq.end()
})

server.listen(PROXY_PORT, () => {
    console.log(`Dev-Proxy läuft auf http://localhost:${PROXY_PORT}`)
    console.log(`  /login, /register, /dashboard, /api/*  → http://${FRONTEND_HOST}:${FRONTEND_PORT}`)
    console.log(`  alles andere                            → http://${LANDING_HOST}:${LANDING_PORT}`)
})