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


## Context: Repository — RETURNING clause harus lengkap

### Kesalahpahaman awal
Dikira `RETURNING` di query `INSERT`/`UPDATE` itu boleh cuma ambil sebagian kolom
seadanya, asal `id`-nya ada. Ternyata masalahnya: `StructScan()` scan hasil
`RETURNING` ke struct `User` penuh — kolom yang gak ikut di-`RETURNING` otomatis
jadi **zero-value** di struct itu (bukan diisi dari DB), meski di tabel aslinya
ada isinya.

```go
// SALAH: email gak ikut RETURNING, jadi Email di struct kosong walau
// email sebenernya kesimpen bener di DB
query := `INSERT INTO users (name, email, password_hash)
    VALUES ($1, $2, $3) RETURNING id, name, role, created_at`

// BENAR: semua kolom yang dibutuhin caller (service/DTO) ikut di-RETURNING
query := `INSERT INTO users (name, email, password_hash)
    VALUES ($1, $2, $3)
    RETURNING id, name, email, role, created_at, updated_at`
```

### Kenapa penting
Struct yang balik dari `Create()`/`Update()` ini sering langsung dipakai buat
convert ke response DTO (`ToUserResponse`). Kalau `RETURNING` gak lengkap,
bug-nya **silent** — gak ada error, cuma field di response API jadi kosong.

### Kolom sensitif gak boleh sembarangan ikut di semua query
`password_hash` sengaja **gak** ikut di-`SELECT`/`RETURNING` pada method biasa
(`GetAll`, `GetByID`, `Create`, `Update`) — biar gak ke-expose meski lupa
convert ke DTO. Tapi ada 1 use case yang butuh dia: proses `Login` (buat
compare bcrypt) dan `Update` (buat pertahanin hash lama kalau user gak ganti
password). Solusinya: bikin method repository terpisah khusus buat itu
(`GetByEmail` include password_hash, `GetPasswordHashByID` cuma select 1
kolom itu doang) — daripada nambahin `password_hash` ke method umum.

### TODO lanjutan
- [ ] Cek migration ada trigger `updated_at` auto-update atau belum. Kalau
      belum ada, query `Update()` udah pasang `updated_at=NOW()` manual.

## Context: Service layer — Register & Login flow

### Pola: cek-dulu vs insert-langsung (TOCTOU)
Buat `Register`, ada 2 pendekatan cegah duplicate email:
1. **Query dulu** (`GetByEmail` sebelum `Create`) — UX bagus, response cepat
   dan jelas ("email sudah terdaftar").
2. **Insert langsung**, biarin `UNIQUE` constraint di DB yang nolak, baru
   parse error code dari driver.

Pendekatan 1 ada celah race condition (TOCTOU — time-of-check to
time-of-use): dua request register bareng, email sama, bisa keduanya lolos
cek "belum ada" sebelum salah satu sempet insert. `UNIQUE` constraint di DB
itu atomic jadi sebenernya itu yang jadi "penjaga" terakhir yang gak pernah
bohong.

**Keputusan buat sekarang:** pakai pendekatan 1 doang dulu (query dulu),
belum handle race condition di step 2. Sengaja — biar gak keburu ribet
sebelum konsep dasarnya settle. Race condition itu jarang kejadian di
skala kecil, jadi masuk kategori "debt yang sadar diambil", bukan bug
kelupaan.

### Login: kenapa 1 pesan error generik buat 2 kasus beda
```go
// email gak ketemu ATAU password salah -> pesan sama: ErrInvalidCredentials
```
Ini best practice keamanan standar: kalau dibedain ("email gak ketemu" vs
"password salah"), penyerang bisa dipakai buat enumerasi akun (nyoba-nyoba
email random, taulah mana yang valid dari pesan errornya beda).

### Kenapa Login belum generate JWT
Token generation butuh keputusan lain yang belum dibahas (naruh secret di
mana, expiry berapa lama, perlu refresh token atau nggak). Diputusin buat
dipisah ke tahap middleware/auth, biar service layer selesai fokus di 1 hal
dulu: verifikasi kredensial valid, return `*User`.

### TODO lanjutan
- [ ] Race condition di `Register` (TOCTOU) — belum ditangani, sadar diambil
      sebagai debt.
- [ ] `Update` di service belum cek apakah email baru udah dipakai user lain
      → kalau iya, bakal nembus ke DB dan meledak jadi 500 generik
      (`UNIQUE constraint`), bukan 409 Conflict yang rapi. Perlu ditambah
      cek `GetByEmail` di `Update`, exclude diri sendiri.
- [ ] Handler pakai `c.Request.Context()` tapi **belum ada timeout** di
      level request — beda sama timeout connection pool yang udah ada duluan.
      Rencana: timeout middleware, deadline ~5-10 detik, nyambung ke
      pembahasan middleware besok.

## Context: JWT Auth + Middleware (Timeout & RBAC)

### Timeout middleware - kenapa perlu wrap ulang c.Request.Context()
`c.Request.Context()` default-nya gak punya batas waktu sama sekali - dia
cuma cancel kalau client disconnect duluan. Kalau query DB hang (bukan
"ditolak" tapi beneran nge-hang, kayak eksperimen network unreachable
sebelumnya), handler bisa nunggu selama-lamanya tanpa batas.

Solusinya: middleware yang wrap context request pakai `context.WithTimeout`,
dipasang global lewat `r.Use(...)`, jalan sebelum handler manapun:

```go
func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{"error": "request timed out"})
		}
	}
}
```

Context baru ini otomatis nurun ke semua layer di bawahnya (service,
repository) karena semua nerima `ctx` yang sama dari `c.Request.Context()`.

### Trade-off: goroutine leak sementara
Kalau timeout kena, goroutine yang jalanin `c.Next()` **gak langsung mati**.
Dia tetep jalan di background sampe operasi DB-nya beneran selesai atau
di-cancel drivernya - meski response `504` udah keburu dikirim ke client.
Bukan infinite leak (bakal berhenti begitu driver Postgres nangkep context
cancellation), tapi ada gap waktu goroutine itu masih "hidup". Ini level
detail yang biasanya baru dioptimasi di production-grade (ada library kayak
`gin-contrib/timeout` yang nanganin lebih matang). Cukup buat tahap belajar
sekarang, penting buat disadari limitasinya.

### JWT: kenapa logic generate/parse dipisah dari middleware
Struktur dibikin 2 layer:
- **`internal/auth`** - `TokenService` (struct, bukan fungsi package-level)
  yang nyimpen secret + expiry, expose `Generate()` dan `Parse()`. Dibikin
  struct biar secret gak jadi global state, dan gampang di-mock kalau nanti
  ada unit test.
- **`internal/middleware`** - `RequireAuth()` makai `TokenService` buat
  verifikasi header `Authorization: Bearer <token>`, taro `user_id` & `role`
  ke `gin.Context` biar bisa diakses handler/middleware berikutnya.

Pemisahan ini biar `auth` package testable sendiri tanpa perlu HTTP context
sama sekali, dan middleware cuma jadi "adapter" yang nyambungin HTTP layer
ke logic auth.

### Kenapa alg confusion check penting di Parse()
```go
if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
    return nil, ErrInvalidToken
}
```
Tanpa cek ini, penyerang bisa nyoba forge token pakai algoritma lain
(misal `none` atau RSA) yang gak tervalidasi bener sama secret HMAC kita.
Validasi signing method itu wajib, bukan opsional.

### RBAC: RequireAuth vs RequireRole - urutan penting
`RequireRole()` **harus** dipasang setelah `RequireAuth()` di middleware
chain, karena dia baca role dari `gin.Context` yang di-set sama
`RequireAuth()`. Kalau kebalik urutannya, `RequireRole` bakal selalu gagal
(gak nemu role di context).

```go
users.Use(middleware.RequireAuth(tokenService))     // set context dulu
users.DELETE("/:id", middleware.RequireRole("admin"), h.Delete) // baru cek role
```

Dites manual: register -> login (dapet token) -> GET /users tanpa token
(401) -> GET /users pake token (200). Semua jalan sesuai ekspektasi.

### TODO lanjutan
- [ ] Test RBAC end-to-end: DELETE /users/:id pakai token role "staff",
      harus dapet 403 Forbidden (bukan 401) - RequireAuth lolos tapi
      RequireRole nolak.
- [ ] PUT /users/:id belum ada cek "user cuma boleh update dirinya sendiri"
      - staff bisa update user lain sekarang, selama dia login. Perlu
      dibandingin id di token (user_id dari context) vs id di URL param.
- [ ] Refresh token belum ada. Token expiry 24 jam, kalau habis user harus
      login ulang dari nol. Perlu dipikirin nanti kalau mau UX lebih baik.
- [ ] JWT_SECRET masih di-export manual tiap buka terminal baru. Rencana:
      pindah ke `.env` + `godotenv` biar gak perlu export ulang tiap sesi.
