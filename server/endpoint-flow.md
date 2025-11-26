# API Endpoint Flow

Dokumen ini menjelaskan alur proses dari beberapa endpoint utama dalam aplikasi game multiplayer.

---

## I. Endpoint: `/create/user`

### Alur:
1. Pengguna memasukkan username melalui form di frontend.
2. Frontend mengirimkan username ke backend melalui endpoint `/create/user`.
3. Backend menyimpan username ke dalam map Players dan mengembalikan response sukses.
4. Frontend menerima response dan menyimpan data yang diperlukan (misalnya playerId) ke localStorage.

---

## II. Endpoint: `/room/create`

### Alur:
1. Pengguna memasukkan username.
   - Frontend mengirimkan username ke backend (jika belum dilakukan).
   - Backend menambahkan user ke dalam map Players.
2. Pengguna memilih jenis permainan yang ingin dimainkan.
   - Frontend mengirimkan request ke endpoint `/room/create` dengan GameType pilihan pengguna.
   - Backend membuat Room baru dengan:
     - GameType sesuai pilihan pengguna.
     - Menambahkan user tersebut ke dalam Room (sebagai Player).
     - Menandai IsActive sebagai false.
   - Backend mengembalikan response berisi roomId dan status IsActive.
3. Frontend menyimpan roomId ke localStorage.
4. Setelah localStorage berisi roomId, komponen permainan yang sesuai (TicTacToeBoard.tsx / ChessBoard.tsx) diaktifkan melalui state.
5. Komponen game akan langsung menginisialisasi koneksi websocket menggunakan useWebSocket dan state yang tersedia.

---

## III. Endpoint: `/room/join`

### Alur:
1. Pengguna memasukkan Room ID untuk bergabung ke room yang sudah ada.
2. Frontend mengirimkan Room ID, GameType, dan Player ID ke endpoint `/room/join`.
3. Backend memproses permintaan dan mengembalikan response yang berisi:
   - Player ID
   - PlayerMark (X/O untuk TicTacToe, White/Black untuk catur)
   - Informasi Room (Room ID, daftar Players, GameState, IsActive, IsAIEnabled)
4. Setelah frontend menerima data Room, komponen permainan (TicTacToeBoard.tsx / ChessBoard.tsx) akan dimuat sesuai GameType.

---

