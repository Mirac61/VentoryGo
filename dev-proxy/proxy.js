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
// Die Asset-Pfade in landing/index.html sind relativ (styles.css, images/...),
// der Browser löst sie also gegen die aufgerufene URL auf — nicht gegen einen
// intern umgeschriebenen Pfad. Deshalb wird "/" per Redirect auf "/landing/"
// geschickt statt serverseitig umgeschrieben: nur mit dem echten Trailing
// Slash in der Adresszeile landet styles.css bei /landing/styles.css.
// Kein <base>-Tag in index.html, das würde in Produktion brechen, wo die
// Landing Page unter "/" liegt.
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
//   / und /landing                          → 302 auf /landing/
//   alles andere                            → :4173 (Landing Page + Assets, unverändert)

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

// Bewusst nur die Formen OHNE Trailing Slash. "/landing/" muss durchgereicht
// werden, sonst redirected der Proxy auf dieselbe URL, die er gerade behandelt.
function isLandingRoot(url) {
    return url === '/' || url === '/landing'
}

function proxyRequest(clientReq, clientRes) {
    if (isLandingRoot(clientReq.url)) {
        clientRes.writeHead(302, { Location: '/landing/' })
        clientRes.end()
        return
    }

    const target = targetFor(clientReq.url)
    proxyTo(clientReq, clientRes, target, clientReq.url)
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
        // Beide Puffer sind der jeweils schon gelesene Rest nach dem Handshake
        // und gehören an die Gegenseite: `head` kommt vom Client und muss zum
        // Upstream, `upstreamHead` umgekehrt. Beim Vite-HMR-Handshake sind sie
        // in der Regel leer — deshalb fällt eine falsche Richtung hier nicht
        // auf, sondern erst bei Traffic, der direkt am Upgrade hängt.
        if (head.length > 0) {
            upstreamSocket.write(head)
        }
        if (upstreamHead.length > 0) {
            clientSocket.write(upstreamHead)
        }
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