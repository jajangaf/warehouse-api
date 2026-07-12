# Learning Notes — warehouse-api

## Context: Background() vs WithTimeout()

### Kesalahpahaman awal
Awalnya dikira `context.Background()` dan `context.WithTimeout()` itu dua pilihan
terpisah ("pake yang ini ATAU yang itu"). Ternyata salah — `WithTimeout()` itu
**membungkus** `Background()`, bukan menggantikannya. Semua fungsi `context.With...`
(WithTimeout, WithCancel, WithValue) butuh parent context, dan `Background()`
biasanya jadi parent paling awal.

```go
// Background() TETAP ada di sini sebagai parent,
// WithTimeout() cuma nambahin aturan "cancel setelah 5 detik"
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

### Kenapa penting: hasil eksperimen nyata
Simulasi: `DB_HOST` diarahkan ke IP yang gak reachable (`10.255.255.1`), 
biar koneksi "hang" (bukan langsung refused).

| Versi context                          | Hasil                                  |
|-----------------------------------------|-----------------------------------------|
| `context.Background()` (tanpa timeout) | Nunggu **1m31s**, harus di-Ctrl+C manual |
| `context.WithTimeout(..., 5*time.Second)` | Gagal otomatis tepat di **5.3s**        |

**Kesimpulan:** tanpa timeout, aplikasi bisa freeze tanpa batas kalau network
bermasalah (bukan cuma "service mati", yang mana errornya cepat karena ada
`connection refused`). Kasus paling bahaya itu network *hang* (firewall silent
drop, host unreachable), bukan ditolak.

### Kapan Background() boleh dipake "telanjang" (tanpa With...)
- Sebagai parent yang mau di-derive lagi (`WithTimeout(Background(), ...)`)
- Context tingkat aplikasi yang emang didesain hidup selama program jalan,
  misal graceful shutdown listener (biasa dikombinasi `signal.NotifyContext`)

### Kapan WAJIB dibungkus timeout
- Semua operasi I/O: koneksi DB, HTTP call ke service lain, query database
- Aturan praktis: startup connection → bungkus timeout di `main.go`.
  Per-request (nanti di handler Gin) → derive dari `c.Request.Context()`,
  bungkus timeout juga sebelum dipake ke service/repository layer.

### TODO lanjutan
- [ ] Load test connection pool pake banyak goroutine + `db.Stats()`
- [ ] Bandingin behavior pool: handler dengan timeout vs tanpa timeout,
      di bawah concurrent load tinggi (nyambung ke rencana testing k6/hey)
