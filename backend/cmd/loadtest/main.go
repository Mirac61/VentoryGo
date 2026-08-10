// Lasttest für POST /api/auth/register: hält -c Clients konstant beschäftigt und
// meldet Durchsatz und Latenz-Perzentile. Jeder Request bekommt eine eigene
// E-Mail, sonst antwortet der Server ab dem zweiten mit 409 und die Messung
// zeigt Konfliktbehandlung statt Passwort-Hashing.
//
// Legt pro Lauf zehntausende Nutzer an. Nur gegen eine Wegwerf-Datenbank
// laufen lassen, Aufräumen übernimmt scripts/loadtest.sh.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// EmailPrefix markiert die angelegten Nutzer, damit das Aufräumen sie
// wiederfindet, ohne echte Konten anzufassen.
const EmailPrefix = "loadtest"

func main() {
	url := flag.String("url", "http://127.0.0.1:8080/api/auth/register", "Endpoint")
	clients := flag.Int("c", 8, "gleichzeitige Clients")
	duration := flag.Duration("d", 12*time.Second, "Laufzeit")
	tag := flag.String("tag", "run", "Kennzeichnung der angelegten E-Mails")
	flag.Parse()

	var (
		mu        sync.Mutex
		latencies []time.Duration
		counter   atomic.Int64
		failed    atomic.Int64
		codes     sync.Map
	)

	deadline := time.Now().Add(*duration)
	start := time.Now()

	var wg sync.WaitGroup
	for worker := range *clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: time.Minute}
			for time.Now().Before(deadline) {
				body := fmt.Sprintf(`{"email":"%s-%s-%d-%d@example.com","password":"correct horse battery"}`,
					EmailPrefix, *tag, worker, counter.Add(1))

				sent := time.Now()
				resp, err := client.Post(*url, "application/json", bytes.NewBufferString(body))
				took := time.Since(sent)
				if err != nil {
					failed.Add(1)
					continue
				}
				resp.Body.Close()

				count, _ := codes.LoadOrStore(resp.StatusCode, new(atomic.Int64))
				count.(*atomic.Int64).Add(1)

				mu.Lock()
				latencies = append(latencies, took)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	elapsed := time.Since(start)
	if len(latencies) == 0 {
		fmt.Fprintln(os.Stderr, "keine Antworten erhalten, laeuft der Server?")
		os.Exit(1)
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	percentile := func(p float64) time.Duration {
		return latencies[int(p*float64(len(latencies)-1))].Round(time.Millisecond)
	}

	statuses := ""
	codes.Range(func(code, count any) bool {
		statuses += fmt.Sprintf(" %d=%d", code, count.(*atomic.Int64).Load())
		return true
	})

	fmt.Printf("clients=%d req=%d fehler=%d req/s=%.1f p50=%v p95=%v p99=%v status:%s\n",
		*clients, len(latencies), failed.Load(),
		float64(len(latencies))/elapsed.Seconds(),
		percentile(0.50), percentile(0.95), percentile(0.99), statuses)
}
