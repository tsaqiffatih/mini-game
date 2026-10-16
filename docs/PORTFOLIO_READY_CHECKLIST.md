# Portfolio-Ready Checklist — v1.0.0

Checklist ini disusun berdasarkan audit kode aktual dan digunakan sebagai **master checklist recovery, stabilization, production preparation, dan portfolio release**.

Target milestone:

> **`v1.0.0 — Portfolio Release`**

Fokus milestone ini adalah membuat aplikasi realtime multiplayer yang sudah ada menjadi **stabil, aman secara fundamental, reproducible, deployable pada single VPS, dan layak dipresentasikan sebagai portfolio**.

Tidak ada fitur game baru atau perluasan arsitektur sebelum P0–P2 selesai.

---

# P0 — RECOVER & BLOCKERS

**Tujuan:** memastikan project kembali dalam kondisi dapat dibangun, dijalankan, dan dipercaya sebelum masuk production preparation.

---

## P0.1 — Baseline Validation

Backend sudah lolos saat audit. Jalankan ulang setelah perubahan yang relevan.

* [x] Backend test (`go test ./...`)
* [x] Backend vet (`go vet ./...`)
* [x] Backend build (`go build ./...`)
* [x] Backend race detector (`go test -race ./...`)
* [x] Frontend lint (`npm run lint`)
* [x] Frontend production build (`npm run build`)

### Definition of Done

```text
Backend test       → PASS
Backend vet        → PASS
Backend build      → PASS
Race detector      → PASS
Frontend lint      → PASS
Frontend build     → PASS
```

---

## P0.2 — Session Ownership & Fundamental Security

* [x] Implementasikan session token yang tidak dapat ditebak saat player dibuat.
* [x] Verifikasi session ownership pada seluruh HTTP endpoint.
* [x] Verifikasi session ownership pada WebSocket connection.
* [x] Jangan hanya mempercayai `player_id` dari request atau query string.
* [x] Pastikan Player A tidak dapat mengakses session Player B.
* [x] Tambahkan regression test untuk session ownership.
* [x] Validasi WebSocket origin berdasarkan origin yang diizinkan.
* [x] Pastikan origin yang tidak diizinkan ditolak.

### Definition of Done

```text
Player A + Token A
        ↓
akses Player A → PASS

Player A + Token B
        ↓
akses Player B → DENIED
```

---

## P0.3 — WebSocket & Room Isolation

* [x] Isolasi registry WebSocket berdasarkan kombinasi `roomID + playerID`.
* [x] Pastikan broadcaster hanya mengirim event ke room yang sesuai.
* [x] Pastikan snapshot Room A tidak pernah dikirim ke koneksi Room B.
* [x] Putuskan dan dokumentasikan apakah satu player boleh aktif di lebih dari satu room.
* [x] Tambahkan test cross-room isolation.
* [x] Pastikan chat juga terisolasi berdasarkan room.
* [x] Pastikan game event tidak dapat bocor antar-room.

### Definition of Done

```text
Room A
 ├── Player 1
 └── Player 2

Room B
 ├── Player 3
 └── Player 4

Event Room A
      ↓
Player 1 + Player 2
      ✓

Player 3 + Player 4
      ✗
```

---

## P0.4 — Reconnection Correctness

* [x] Selaraskan frontend retry strategy dengan backend reconnect grace period.
* [x] Tentukan satu source of truth untuk reconnect grace period.
* [x] Dokumentasikan reconnect policy.
* [x] Test reconnect sebelum grace period berakhir.
* [x] Test reconnect tepat/sekitar batas grace period.
* [x] Test player removal setelah grace period berakhir.
* [x] Pastikan stale disconnect cleanup tidak menghapus player yang sudah reconnect.
* [x] Pastikan game state tetap konsisten setelah reconnect.
* [x] Pastikan chat tetap bekerja setelah reconnect.
* [x] Pastikan game move tetap bekerja setelah reconnect.

### Definition of Done

```text
disconnect
    ↓
reconnect within grace
    ↓
session recovered
    ↓
game continues

disconnect
    ↓
grace expires
    ↓
player removed
```

---

## P0.5 — Server-Authoritative Game State

* [x] Pastikan seluruh game mutation dilakukan melalui `Room`.
* [x] Pastikan handler tidak memodifikasi `GameState` secara langsung.
* [x] Pastikan client tidak dapat menentukan authoritative game state.
* [x] Validasi game action di server sebelum state berubah.
* [x] Tambahkan regression test untuk invalid/illegal game action.
* [x] Pastikan Chess hanya menerima legal move.
* [x] Pastikan Tic-Tac-Toe hanya menerima valid move.
* [x] Pastikan client menerima authoritative snapshot dari server.

---

## P0.6 — State Version & Stale Snapshot Protection

* [x] Pastikan `state_version` digunakan secara konsisten pada realtime state.
* [x] Pastikan snapshot lama tidak dapat menimpa state terbaru.
* [x] Tambahkan regression test untuk stale snapshot.
* [x] Pastikan reconnect tidak menyebabkan state lama meng-overwrite state terbaru.

### Definition of Done

```text
state_version 10
       ↓
state_version 11
       ↓
snapshot #10 datang terlambat
       ↓
REJECT
```

---

## P0.7 — Stockfish Reproducibility

* [x] Jadikan binary Stockfish reproducible dari clean Git clone.
* [x] Pastikan Dockerfile tidak bergantung pada binary lokal yang di-ignore Git.
* [x] Pastikan clean Docker build dapat menyediakan Stockfish.
* [x] Tangani kegagalan Stockfish dengan error yang jelas pada UI/API.
* [x] Uji Chess AI menggunakan container hasil clean build.

### Definition of Done

```text
clean clone
    ↓
docker build --no-cache
    ↓
Stockfish tersedia
    ↓
Chess AI works
```

---

# 🚦 P0 GATE — BLOCKER RELEASE

**Jangan masuk P1 sebelum semua blocker berikut selesai.**

* [x] Backend test/vet/build/race PASS
* [x] Frontend lint/build PASS
* [x] Session ownership enforced
* [x] WebSocket origin validation PASS
* [x] Cross-room isolation PASS
* [x] Reconnection policy PASS
* [x] Server-authoritative mutation PASS
* [x] Stale state protection PASS
* [x] Stockfish reproducible

> Jika salah satu blocker gagal, tetap di P0.

---

# P1 — STABILIZE & PRODUCTION PREPARATION

**Tujuan:** setelah core correctness aman, siapkan aplikasi agar layak dijalankan secara production-like.

---

## P1.1 — Input Validation

* [x] Validasi player ID.
* [x] Validasi room ID.
* [x] Validasi player name.
* [x] Validasi chat message length.
* [x] Validasi request body.
* [x] Tambahkan batas ukuran HTTP request body.
* [x] Tolak unknown JSON fields bila sesuai.
* [x] Validasi payload WebSocket.
* [x] Tangani malformed JSON dengan error yang konsisten.
* [x] Validasi room code.
* [x] Tangani room code collision.
* [x] Evaluasi penggunaan `crypto/rand` untuk room code.

---

## P1.2 — Rate Limiting & Abuse Protection

* [x] Perbaiki rate limiter agar tidak menggunakan `RemoteAddr` mentah yang mencakup port.
* [x] Gunakan client identity yang sesuai untuk rate limiting.
* [x] Pastikan rate limiter tidak mengganggu reconnect normal.
* [x] Tambahkan basic protection untuk endpoint/WebSocket yang relevan.
* [x] Tidak perlu membuat rate limiting enterprise-grade.

---

## P1.3 — Chess AI Resource Protection

* [x] Batasi jumlah Chess AI room/process aktif untuk single VPS.
* [x] Pastikan AI process selalu dihentikan ketika room selesai/dihapus.
* [x] Pastikan error Stockfish tidak meninggalkan process/resource.
* [x] Uji beberapa AI room secara bersamaan.
* [x] Pastikan resource limit mencegah spawning tidak terbatas.

---

## P1.4 — Health Check & Operational Readiness

* [x] Tambahkan endpoint `/healthz`.
* [x] Pastikan `/healthz` mengembalikan status yang benar.
* [x] Tambahkan Docker health check.
* [x] Verifikasi graceful shutdown.
* [x] Verifikasi behavior ketika container/server restart.
* [x] Pastikan resource seperti Stockfish dibersihkan saat shutdown.

---

## P1.5 — Logging Cleanup

* [x] Hapus debug logging.
* [x] Hapus logging sementara untuk debugging.
* [x] Hapus log seperti `<<<<<<<<<`.
* [x] Pertahankan logging yang relevan untuk production.
* [x] Pastikan session token tidak masuk log.
* [x] Pastikan secret tidak masuk log.
* [x] Pastikan error production tidak membocorkan informasi internal secara berlebihan.

---

## P1.6 — Dead Code & Dependency Cleanup

* [x] Bersihkan dead code.
* [x] Bersihkan commented-out code.
* [x] Hapus utility yang tidak digunakan.
* [x] Hapus dependency yang tidak digunakan.
* [x] Evaluasi `ErrorBoundary`.
* [x] Evaluasi `parseCookies`.
* [x] Evaluasi `react-chessboard`.
* [x] Putuskan migration atau penghapusan alias API Chess yang deprecated.
* [x] Jangan melakukan refactor architecture besar.

> **Rule:** cleanup boleh, redesign besar tidak.

---

## P1.7 — Critical Tests

### Backend

* [x] Session ownership
* [x] Cross-room isolation
* [x] Reconnect
* [x] Stale disconnect cleanup
* [x] Room lifecycle
* [x] Invalid input
* [x] Invalid game action
* [x] State version
* [x] AI resource limit

### Manual integration

* [x] Player A create room
* [x] Player B join room
* [x] Multiplayer game
* [x] Chat
* [x] Disconnect
* [x] Reconnect
* [x] Game reset
* [x] AI Chess

---

## P1.8 — Docker & Production Environment

* [x] Docker image dapat dibangun dari clean clone.
* [x] `docker build --no-cache` berhasil.
* [x] `.env.example` tersedia untuk backend.
* [x] `.env.example` tersedia untuk frontend bila diperlukan.
* [x] Tidak ada secret di `.env.example`.
* [x] Dokumentasikan seluruh environment variable.
* [x] Siapkan production environment variables.

Contoh:

```text
ALLOWED_ORIGINS=https://...
NEXT_PUBLIC_HTTP_BACKEND_URL=https://...
NEXT_PUBLIC_WS_BACKEND_URL=wss://...
```

---

## P1.9 — Caddy / HTTPS / WSS

* [x] Siapkan Caddy production configuration.
* [x] Konfigurasi domain production.
* [x] Konfigurasi HTTPS.
* [x] Konfigurasi WebSocket/WSS.
* [x] Pastikan reverse proxy meneruskan WebSocket dengan benar.
* [x] Pastikan `ALLOWED_ORIGINS` sesuai dengan frontend production.
* [x] Pastikan HTTP/HTTPS endpoint bekerja.
* [x] Pastikan WSS connection berhasil.

---

# 🚦 P1 GATE — PRODUCTION-LIKE READY

Sebelum masuk P2:

* [x] Application dapat dijalankan via Docker.
* [x] `/healthz` PASS.
* [x] HTTPS PASS.
* [x] WSS PASS.
* [x] Multiplayer PASS.
* [x] Chat PASS.
* [x] Reconnect PASS.
* [x] Chess AI PASS.
* [x] Graceful shutdown PASS.
* [x] Tidak ada secret/debug artifact.
* [x] Resource AI terkendali.

Targetnya:

```text
Browser A
     ↕
 HTTPS/WSS
     ↕
 Caddy
     ↕
 Docker
     ↕
 Go Backend
     ↕
 Room
```

---

# P2 — PORTFOLIO POLISH

**Tujuan:** bukan menambah kemampuan aplikasi, tetapi membuat hasil engineering-mu mudah dipahami dan dipresentasikan.

---

## P2.1 — README

* [x] Tulis ulang README root berdasarkan kondisi kode aktual.
* [x] Ganti README frontend template Next.js.
* [x] Jelaskan project secara singkat.
* [x] Jelaskan fitur.
* [x] Jelaskan tech stack.
* [x] Jelaskan local development.
* [x] Jelaskan environment variables.
* [x] Jelaskan Docker.
* [x] Jelaskan deployment.
* [x] Jelaskan known limitations.
* [x] Cantumkan demo URL.
* [x] Cantumkan repository information.

---

## P2.2 — Architecture Documentation

* [x] Buat architecture diagram sederhana.
* [x] Dokumentasikan Frontend → Backend → Room → Game Engine.
* [x] Dokumentasikan WebSocket flow.
* [x] Dokumentasikan server-authoritative state.
* [x] Dokumentasikan reconnect mechanism.
* [x] Dokumentasikan `state_version`.
* [x] Dokumentasikan Room sebagai owner game state.
* [x] Dokumentasikan single-instance architecture.

---

## P2.3 — Engineering Decisions

README/documentation menjelaskan alasan:

* [x] Mengapa menggunakan WebSocket.
* [x] Mengapa server-authoritative.
* [x] Mengapa `Room` menjadi owner state.
* [x] Mengapa menggunakan mutex.
* [x] Mengapa menggunakan `state_version`.
* [x] Mengapa reconnect memiliki grace period.
* [x] Mengapa menggunakan in-memory state.
* [x] Mengapa belum menggunakan database.
* [x] Apa trade-off single-instance architecture.

---

## P2.4 — CI

* [x] GitHub Actions backend test.
* [x] GitHub Actions backend vet.
* [x] GitHub Actions backend build.
* [x] GitHub Actions frontend lint.
* [x] GitHub Actions frontend build.
* [x] Pastikan CI berjalan pada clean environment.

### Optional

* [x Manual test] Frontend/E2E test untuk critical user flow.

> E2E **tidak menjadi blocker v1.0** apabila manual verification sudah lengkap.

---

## P2.5 — Demo UX

* [x] Landing page dapat digunakan tanpa kebingungan.
* [x] Create room berhasil.
* [x] Room code mudah ditemukan/copy.
* [x] Join room berhasil.
* [x] Loading state jelas.
* [x] Connection state jelas.
* [x] Reconnecting state jelas.
* [x] Room expired ditangani dengan jelas.
* [x] Invalid session ditangani dengan jelas.
* [x] Server unavailable ditangani dengan jelas.
* [x] Tidak ada error state yang hanya menampilkan pesan generik seperti `error`.

---

## P2.6 — Repository Hygiene

* [x] Tidak ada `.env` yang ter-commit.
* [x] Tidak ada secret yang ter-commit.
* [x] Tidak ada binary lokal yang seharusnya tidak di-track.
* [x] Tidak ada Stockfish binary ilegal/accidental di repository.
* [x] `.gitignore` sesuai.
* [x] Repository tidak memiliki debug artifacts.
* [x] Git status bersih sebelum release.

---

# 🚦 FINAL GATE — PORTFOLIO RELEASE

Sebelum `v1.0.0`, lakukan **full manual verification**.

## Multiplayer

* [x] Browser A create room.
* [x] Browser B join room.
* [x] Tic-Tac-Toe multiplayer.
* [x] Tic-Tac-Toe chat.
* [x] Tic-Tac-Toe reset.
* [x] Tic-Tac-Toe reconnect.
* [x] Chess multiplayer.
* [x] Chess legal move validation.
* [x] Chess chat.
* [x] Chess reset.
* [x] Chess reconnect.
* [x] Chess AI.

## Production

* [ ] HTTPS.
* [ ] WSS.
* [ ] `/healthz`.
* [ ] Docker.
* [ ] Stockfish container.
* [ ] Graceful shutdown.
* [ ] Restart behavior.

## Security

* [x] Session ownership.
* [x] Origin validation.
* [ ] Cross-room isolation.
* [x] Input validation.
* [x] Basic rate limiting.

## Quality

* [x] `go test ./...`
* [x] `go test -race ./...`
* [x] `go vet ./...`
* [x] `go build ./...`
* [x] `npm run lint`
* [x] `npm run build`

## Portfolio

* [x] README selesai.
* [x] Architecture diagram selesai.
* [ ] Deployment runbook selesai.
* [ ] Known limitations terdokumentasi.
* [ ] Demo URL aktif.
* [ ] Repository siap dibagikan.

---

# 🛑 v1.0.0 RELEASE

Jika seluruh **Final Gate** selesai:

```text
v1.0.0 — Portfolio Release
```

**STOP.**

Jangan menambahkan fitur baru sebelum milestone berikutnya dibuat secara eksplisit.

---

# Explicitly Out of Scope

Hal-hal berikut **tidak diperlukan untuk `v1.0.0`**:

* [ ] Database persistence.
* [ ] Redis.
* [ ] Kubernetes.
* [ ] Multi-instance scaling.
* [ ] Distributed WebSocket state.
* [ ] Worker pool kompleks.
* [ ] OAuth.
* [ ] Ranking.
* [ ] Leaderboard.
* [ ] Tournament.
* [ ] Match history.
* [ ] Spectator mode.
* [ ] Game baru.
* [ ] Major UI redesign.
* [ ] Enterprise-grade rate limiting.
* [ ] Enterprise authentication system.

Setelah `v1.0.0` live, barulah fitur-fitur tersebut dapat dipertimbangkan sebagai **P3 / post-release roadmap**.

---

# Urutan Eksekusi

Yang paling penting: **jangan mengerjakan checklist ini seperti daftar TODO acak.**

Urutannya:

```text
                 CURRENT REPO
                      │
                      ▼
             ┌─────────────────┐
             │ P0.1 Baseline   │
             └────────┬────────┘
                      ▼
             ┌─────────────────┐
             │ P0.2 Security   │
             └────────┬────────┘
                      ▼
             ┌─────────────────┐
             │ P0.3 Realtime   │
             └────────┬────────┘
                      ▼
             ┌─────────────────┐
             │ P0.4 Reconnect  │
             └────────┬────────┘
                      ▼
             ┌─────────────────┐
             │ P0.5–P0.7      │
             │ Correctness/AI  │
             └────────┬────────┘
                      ▼
                  🚦 P0 GATE
                      │
                      ▼
             ┌─────────────────┐
             │      P1         │
             │ Stabilization   │
             └────────┬────────┘
                      ▼
                  🚦 P1 GATE
                      │
                      ▼
             ┌─────────────────┐
             │      P2         │
             │ Portfolio Polish│
             └────────┬────────┘
                      ▼
                🚦 FINAL GATE
                      │
                      ▼
                  v1.0.0
                      │
                      ▼
                   🛑 STOP
```

---

## Execution Rule

**Jangan mengerjakan item lintas fase sebelum gate fase sebelumnya terpenuhi**, kecuali sebuah task memang diperlukan sebagai dependency untuk menyelesaikan gate tersebut.

Checklist awal tidak perlu dibuang. Isinya tetap menjadi baseline recovery; yang berubah adalah pengelompokan menjadi **fase, dependency, gate, dan regression test** agar pengerjaan tidak dilakukan secara acak dan milestone `v1.0.0` memiliki batas selesai yang jelas.
