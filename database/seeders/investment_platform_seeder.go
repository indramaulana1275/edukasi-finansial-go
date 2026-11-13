package seeders

import (
	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

type InvestmentPlatformSeeder struct{}

// Signature The name and signature of the seeder.
func (s *InvestmentPlatformSeeder) Signature() string {
	return "InvestmentPlatformSeeder"
}

// Run mengisi daftar platform investasi awal. Seeder ini hanya menambah
// nama yang belum ada, sehingga teks yang sudah diedit admin tidak ditimpa
// setiap kali db:seed dijalankan.
func (s *InvestmentPlatformSeeder) Run() error {
	existing := make([]models.InvestmentPlatform, 0)
	if err := facades.Orm().Query().Get(&existing); err != nil {
		return err
	}

	byName := make(map[string]bool, len(existing))
	for _, platform := range existing {
		byName[platform.Name] = true
	}

	create := make([]models.InvestmentPlatform, 0)
	for _, seed := range investmentPlatformSeeds() {
		if byName[seed.Name] {
			continue
		}

		create = append(create, models.InvestmentPlatform{
			Name:        seed.Name,
			Category:    seed.Category,
			Type:        seed.Type,
			LogoDomain:  seed.LogoDomain,
			Description: seed.Description,
			Pros:        models.PlatformStringsToJSON(seed.Pros),
			Cons:        models.PlatformStringsToJSON(seed.Cons),
		})
	}

	if len(create) == 0 {
		return nil
	}

	return facades.Orm().Query().Create(&create)
}

// platformSeed adalah bentuk mentah satu platform sebelum kolom pros/cons
// diubah menjadi JSON.
type platformSeed struct {
	Name        string
	Category    string
	Type        string
	LogoDomain  string
	Description string
	Pros        []string
	Cons        []string
}

func investmentPlatformSeeds() []platformSeed {
	return []platformSeed{
		{
			Name:        "Indodax",
			Category:    "crypto",
			Type:        "Lokal",
			LogoDomain:  "indodax.com",
			Description: "Bursa kripto lokal tertua di Indonesia dan terdaftar di Bappebti, dengan seluruh transaksi memakai pasangan rupiah.",
			Pros: []string{
				"Deposit dan penarikan rupiah langsung dari bank lokal",
				"Fee trading sekitar 0,3% per transaksi",
				"Likuiditas tinggi untuk koin-koin besar",
			},
			Cons: []string{
				"Tampilan penuh angka terasa rumit bagi pemula",
				"Biaya penarikan kripto ke dompet sendiri cukup mahal",
				"Verifikasi identitas wajib sebelum bisa bertransaksi",
			},
		},
		{
			Name:        "Tokocrypto",
			Category:    "crypto",
			Type:        "Lokal",
			LogoDomain:  "tokocrypto.com",
			Description: "Bursa kripto lokal berizin Bappebti dengan tampilan yang lebih sederhana untuk pembeli pertama.",
			Pros: []string{
				"Aplikasi ringkas untuk pemula",
				"Fee trading mulai 0,1% per transaksi",
				"Pilihan koin cukup banyak dengan pasangan rupiah",
			},
			Cons: []string{
				"Likuiditas masih di bawah bursa global",
				"Fee tersembunyi pada selisih harga (spread) koin kecil",
				"Biaya penarikan mengikuti tarif jaringan yang bisa naik",
			},
		},
		{
			Name:        "Pintu",
			Category:    "crypto",
			Type:        "Lokal",
			LogoDomain:  "pintu.co.id",
			Description: "Aplikasi kripto lokal yang menekankan kemudahan, cocok untuk pembeli pertama yang baru mencoba nominal kecil.",
			Pros: []string{
				"Antarmuka paling mudah dibaca di kelasnya",
				"Tidak ada biaya administrasi bulanan",
				"Materi edukasi tersedia langsung di aplikasi",
			},
			Cons: []string{
				"Biaya transaksi relatif tinggi, sekitar 0,5%",
				"Pilihan koin lebih sedikit daripada bursa besar",
				"Kunci aset dikelola platform, bukan disimpan sendiri",
			},
		},
		{
			Name:        "Reku",
			Category:    "crypto",
			Type:        "Lokal",
			LogoDomain:  "reku.id",
			Description: "Platform kripto lokal berizin Bappebti yang memadukan jual-beli koin, marketplace NFT, dan komunitas belajar.",
			Pros: []string{
				"Fee trading cenderung rendah untuk pasangan rupiah",
				"Tidak ada biaya administrasi bulanan",
				"Tampilan modern dan ringan dipakai pemula",
			},
			Cons: []string{
				"Biaya penarikan ke dompet eksternal relatif mahal",
				"Fee tersembunyi pada selisih harga koin kecil",
				"Pilihan koin lebih sedikit dari bursa global",
			},
		},
		{
			Name:        "Nanovest",
			Category:    "crypto",
			Type:        "Lokal",
			LogoDomain:  "nanovest.io",
			Description: "Aplikasi lokal yang menggabungkan kripto, saham Amerika, dan emas dalam satu akun dengan nominal kecil.",
			Pros: []string{
				"Bisa mulai dari nominal kecil, termasuk saham AS",
				"Tidak ada biaya administrasi bulanan",
				"Satu aplikasi untuk kripto, saham, dan emas",
			},
			Cons: []string{
				"Fee tersembunyi pada selisih harga (spread) transaksi",
				"Pilihan aset lebih terbatas daripada platform khusus",
				"Riwayat operasional platform masih relatif pendek",
			},
		},
		{
			Name:        "Pluang (Crypto)",
			Category:    "crypto",
			Type:        "Lokal",
			LogoDomain:  "pluang.com",
			Description: "Sisi kripto dari aplikasi Pluang: pembelian koin langsung dengan rupiah, berdampingan dengan emas dan reksa dana.",
			Pros: []string{
				"Belanja kripto bisa dimulai dari nominal sangat kecil",
				"Tidak ada biaya administrasi bulanan",
				"Satu aplikasi untuk kripto, emas, dan reksa dana",
			},
			Cons: []string{
				"Fee tersembunyi pada selisih harga (spread) koin",
				"Pilihan koin lebih sedikit daripada bursa khusus kripto",
				"Pengguna tidak memegang kunci pribadi asetnya",
			},
		},
		{
			Name:        "Bittime",
			Category:    "crypto",
			Type:        "Lokal",
			LogoDomain:  "bittime.com",
			Description: "Bursa kripto lokal berizin Bappebti dengan harga transparan dan akademi edukasi untuk investor baru.",
			Pros: []string{
				"Fee trading transparan dan kompetitif",
				"Deposit rupiah dari bank tertentu tanpa biaya",
				"Ada fitur copy trading dan akademi edukasi",
			},
			Cons: []string{
				"Volume perdagangan masih di bawah bursa besar",
				"Biaya penarikan kripto mengikuti tarif jaringan",
				"Pilihan produk turunan (derivatif) terbatas",
			},
		},
		{
			Name:        "Triv",
			Category:    "crypto",
			Type:        "Lokal",
			LogoDomain:  "triv.id",
			Description: "Aplikasi lokal yang menyatukan jual-beli kripto dan emas digital, dengan alur pendaftaran yang cepat.",
			Pros: []string{
				"Bisa mulai dari nominal kecil tanpa biaya buka akun",
				"Tidak ada biaya administrasi bulanan",
				"Proses daftar dan verifikasi cepat",
			},
			Cons: []string{
				"Fee tersembunyi pada selisih harga beli dan jual",
				"Pilihan koin yang tersedia masih terbatas",
				"Belum ada alat analisis teknikal yang dalam",
			},
		},
		{
			Name:        "Binance",
			Category:    "crypto",
			Type:        "Luar Negeri",
			LogoDomain:  "binance.com",
			Description: "Bursa kripto terbesar di dunia dengan pilihan koin dan fitur paling lengkap, tetapi tidak berizin di Indonesia.",
			Pros: []string{
				"Likuiditas terbesar dengan spread paling tipis",
				"Fee trading 0,1% dan bisa turun lagi dengan BNB",
				"Fitur paling lengkap: spot, futures, hingga earn",
			},
			Cons: []string{
				"Tidak terdaftar di Bappebti, tanpa perlindungan regulator lokal",
				"Biaya penarikan (withdrawal) cukup mahal",
				"Antarmuka sangat padat untuk pemula",
			},
		},
		{
			Name:        "Coinbase",
			Category:    "crypto",
			Type:        "Luar Negeri",
			LogoDomain:  "coinbase.com",
			Description: "Bursa asal Amerika Serikat yang sahamnya tercatat di bursa efek dan dikenal paling ketat soal kepatuhan.",
			Pros: []string{
				"Regulasi AS yang jelas dan perusahaan publik",
				"Tidak ada biaya administrasi bulanan",
				"Dana pengguna dipisahkan dan diaudit",
			},
			Cons: []string{
				"Fee transaksi jauh lebih mahal, bisa 1-2% per order",
				"Pilihan koin lebih sedikit",
				"Verifikasi identitas memerlukan dokumen internasional",
			},
		},
		{
			Name:        "OKX",
			Category:    "crypto",
			Type:        "Luar Negeri",
			LogoDomain:  "okx.com",
			Description: "Bursa kripto global dengan fitur trading lengkap dan likuiditas besar di pasar spot maupun derivatif.",
			Pros: []string{
				"Fee trading rendah, mulai 0,08% per transaksi",
				"Likuiditas besar di pasar spot dan futures",
				"Ada dompet Web3 bawaan untuk menyimpan aset sendiri",
			},
			Cons: []string{
				"Tidak berizin di Indonesia",
				"Biaya deposito internasional dan konversi mata uang tinggi",
				"Struktur biaya derivatif bisa membingungkan pemula",
			},
		},
		{
			Name:        "Bybit",
			Category:    "crypto",
			Type:        "Luar Negeri",
			LogoDomain:  "bybit.com",
			Description: "Bursa kripto global yang populer karena antarmuka bersih dan pasar derivatifnya yang dalam.",
			Pros: []string{
				"Fee maker rendah, bahkan 0% untuk beberapa pasangan",
				"Antarmuka modern dan relatif mudah dipahami",
				"Likuiditas futures termasuk yang terbesar",
			},
			Cons: []string{
				"Tidak diawasi Bappebti maupun OJK",
				"Biaya penarikan berbeda-beda antar jaringan",
				"Fitur leverage tinggi berisiko bagi pemula",
			},
		},
		{
			Name:        "Kraken",
			Category:    "crypto",
			Type:        "Luar Negeri",
			LogoDomain:  "kraken.com",
			Description: "Bursa asal Amerika Serikat yang dikenal lama soal keamanan dan rekam jejak kepatuhannya.",
			Pros: []string{
				"Reputasi keamanan dengan bukti cadangan aset terbuka",
				"Fee trading mulai 0,16% dan turun seiring volume",
				"Deposit fiat dari beberapa bank internasional",
			},
			Cons: []string{
				"Biaya deposito internasional tinggi",
				"Tidak menyediakan pasangan rupiah",
				"Antarmuka profesional terasa rumit untuk pemula",
			},
		},
		{
			Name:        "KuCoin",
			Category:    "crypto",
			Type:        "Luar Negeri",
			LogoDomain:  "kucoin.com",
			Description: "Bursa global dengan daftar koin yang sangat luas, sering menjadi tempat pertama koin baru diperdagangkan.",
			Pros: []string{
				"Fee trading 0,1% dan lebih murah untuk maker",
				"Banyak koin kecil yang belum ada di bursa lain",
				"Ada program earn dan bot trading",
			},
			Cons: []string{
				"Tidak ada izin regulator Indonesia",
				"Biaya penarikan token kecil relatif mahal",
				"Risiko koin baru lebih tinggi dan butuh riset ekstra",
			},
		},
		{
			Name:        "Bitget",
			Category:    "crypto",
			Type:        "Luar Negeri",
			LogoDomain:  "bitget.com",
			Description: "Bursa global yang fokus pada copy trading dan pasar derivatif dengan komunitas trader yang besar.",
			Pros: []string{
				"Fee trading spot mulai 0,1%",
				"Fitur copy trading mudah diikuti pemula",
				"Likuiditas derivatif cukup besar",
			},
			Cons: []string{
				"Tidak terdaftar di regulator Indonesia",
				"Biaya pendanaan (funding) futures bisa menggerus posisi",
				"Biaya penarikan mengikuti tarif jaringan yang berubah",
			},
		},
		{
			Name:        "Ajaib",
			Category:    "saham",
			Type:        "Lokal",
			LogoDomain:  "ajaib.co.id",
			Description: "Broker saham lokal berizin OJK dengan biaya transaksi rendah dan aplikasi yang ditujukan untuk investor ritel.",
			Pros: []string{
				"Terdaftar dan diawasi OJK",
				"Fee beli mulai 0,1% dari nilai transaksi",
				"Saham dan reksa dana bisa dibeli dari satu aplikasi",
			},
			Cons: []string{
				"Fee jual lebih tinggi, sekitar 0,2%",
				"Sebagian data riset hanya untuk akun premium",
				"Pembukaan rekening butuh verifikasi video",
			},
		},
		{
			Name:        "Bibit",
			Category:    "saham",
			Type:        "Lokal",
			LogoDomain:  "bibit.id",
			Description: "Aplikasi reksa dana dan saham bagi investor baru yang ingin mulai rutin menabung dengan nominal kecil.",
			Pros: []string{
				"Pembelian reksa dana tanpa biaya transaksi",
				"Reksa dana bisa dimulai dari Rp10.000",
				"Ada Robo Advisor yang menyusun portofolio otomatis",
			},
			Cons: []string{
				"Fee jual saham lebih tinggi daripada broker khusus",
				"Pilihan saham dan risetnya terbatas",
				"Data pasar tidak sedalam platform khusus saham",
			},
		},
		{
			Name:        "Stockbit",
			Category:    "saham",
			Type:        "Lokal",
			LogoDomain:  "stockbit.com",
			Description: "Platform sosial saham dengan data fundamental lengkap, terhubung ke broker Stockbit Sekuritas untuk eksekusi transaksi.",
			Pros: []string{
				"Laporan keuangan dan rasio perusahaan lengkap",
				"Fee transaksi kompetitif lewat sekuritas terafiliasi",
				"Ada komunitas dan fitur diskusi saham",
			},
			Cons: []string{
				"Data analisis terdalam dikenai biaya langganan bulanan",
				"Terlalu banyak informasi bagi pemula",
				"Fokus hanya pada saham Indonesia",
			},
		},
		{
			Name:        "Mirae Asset",
			Category:    "saham",
			Type:        "Lokal",
			LogoDomain:  "sekuritas.miraeasset.co.id",
			Description: "Sekuritas berizin OJK dengan jangkauan riset luas, dukungan reksa dana, dan sistem autopilot.",
			Pros: []string{
				"Terdaftar dan diawasi OJK",
				"Fee transaksi bersaing serta ada promo untuk akun baru",
				"Riset analis dan reksa dana tersedia dalam satu akun",
			},
			Cons: []string{
				"Fee jual tetap berlaku meski nominal transaksinya kecil",
				"Tampilan aplikasi terasa lebih kaku untuk pemula",
				"Pembukaan rekening perlu verifikasi video call",
			},
		},
		{
			Name:        "IPOT (Indo Premier)",
			Category:    "saham",
			Type:        "Lokal",
			LogoDomain:  "indopremier.com",
			Description: "Aplikasi milik Indo Premier Sekuritas, salah satu broker online pertama di Indonesia dengan komunitas investor aktif.",
			Pros: []string{
				"Terdaftar dan diawasi OJK",
				"Tidak ada biaya administrasi bulanan",
				"Komunitas dan kelas edukasi gratis",
			},
			Cons: []string{
				"Fee transaksi mengikuti tarif standar broker",
				"Biaya transfer dana ke bank tertentu masih berlaku",
				"Antarmuka dinilai ketinggalan dibanding aplikasi baru",
			},
		},
		{
			Name:        "BNI Sekuritas",
			Category:    "saham",
			Type:        "Lokal",
			LogoDomain:  "bnisekuritas.co.id",
			Description: "Sekuritas milik BNI yang memudahkan nasabah bank bertransaksi saham lewat aplikasi BIONS.",
			Pros: []string{
				"Terdaftar dan diawasi OJK",
				"Terhubung rekening BNI tanpa biaya transfer RDN",
				"Ada layanan riset dan reksa dana",
			},
			Cons: []string{
				"Fee transaksi standar, kurang cocok untuk trader aktif",
				"Aplikasi terasa lebih rumit daripada broker digital baru",
				"Fee tambahan berlaku di luar bank BNI",
			},
		},
		{
			Name:        "Mandiri Sekuritas",
			Category:    "saham",
			Type:        "Lokal",
			LogoDomain:  "mandirisekuritas.co.id",
			Description: "Sekuritas bank BUMN dengan platform MOST yang menyatukan saham, reksa dana, dan obligasi ritel.",
			Pros: []string{
				"Terdaftar dan diawasi OJK",
				"Fee transaksi kompetitif serta sering ada promo reksa dana",
				"Transaksi obligasi ritel (SBN) bisa dari aplikasi yang sama",
			},
			Cons: []string{
				"Biaya penarikan dana ke bank lain dapat berlaku",
				"Antarmuka MOST punya kurva belajar cukup curam",
				"Data real-time penuh terbatas untuk akun dasar",
			},
		},
		{
			Name:        "Sinarmas Sekuritas",
			Category:    "saham",
			Type:        "Lokal",
			LogoDomain:  "sinarmassekuritas.co.id",
			Description: "Sekuritas berizin OJK yang menyediakan transaksi saham, reksa dana, dan obligasi lewat platform Simas Online.",
			Pros: []string{
				"Terdaftar dan diawasi OJK",
				"Tidak ada biaya administrasi bulanan untuk akun saham",
				"Ada riset harian dan layanan reksa dana",
			},
			Cons: []string{
				"Fee transaksi tidak termasuk yang paling murah",
				"Aplikasi dinilai kurang modern dibanding pemain digital",
				"Pilihan alat analisis teknikal terbatas",
			},
		},
		{
			Name:        "Phillip Sekuritas",
			Category:    "saham",
			Type:        "Lokal",
			LogoDomain:  "phillip.co.id",
			Description: "Sekuritas dengan jaringan internasional yang menyediakan akses saham Indonesia dan beberapa bursa Asia.",
			Pros: []string{
				"Terdaftar dan diawasi OJK",
				"Akses ke bursa regional seperti Singapura dan Hong Kong",
				"Fee transaksi kompetitif untuk pasar Asia",
			},
			Cons: []string{
				"Biaya konversi mata uang untuk transaksi lintas negara",
				"Minimum setoran awal lebih tinggi",
				"Antarmuka dua bahasa tetap terasa rumit untuk pemula",
			},
		},
		{
			Name:        "Interactive Brokers",
			Category:    "saham",
			Type:        "Luar Negeri",
			LogoDomain:  "interactivebrokers.com",
			Description: "Broker global yang sahamnya publik, memberi akses ke bursa di puluhan negara dengan biaya kompetitif.",
			Pros: []string{
				"Akses bursa Amerika, Eropa, dan Asia dari satu akun",
				"Fee transaksi sangat rendah, sebagian saham AS tanpa komisi",
				"Satu akun bisa memegang banyak mata uang",
			},
			Cons: []string{
				"Biaya inaktif dan langganan data pasar berbayar",
				"Deposit rupiah perlu dikonversi dulu dengan biaya kurs",
				"Antarmuka kompleks, jelas bukan untuk pemula",
			},
		},
		{
			Name:        "Gotrade",
			Category:    "saham",
			Type:        "Luar Negeri",
			LogoDomain:  "heygotrade.com",
			Description: "Aplikasi berbahasa Indonesia untuk membeli saham Amerika Serikat dalam pecahan, mulai dari satu dolar.",
			Pros: []string{
				"Bisa membeli saham AS mulai dari 1 dolar",
				"Tanpa komisi per transaksi untuk order dasar",
				"Aplikasi sederhana dan berbahasa Indonesia",
			},
			Cons: []string{
				"Fee tersembunyi pada spread harga dan konversi mata uang",
				"Risiko kurs karena aset dihitung dalam dolar",
				"Pilihan hanya saham dan ETF Amerika",
			},
		},
		{
			Name:        "Charles Schwab",
			Category:    "saham",
			Type:        "Luar Negeri",
			LogoDomain:  "schwab.com",
			Description: "Broker besar asal Amerika Serikat dengan komisi nol untuk saham dan ETF AS serta layanan riset yang luas.",
			Pros: []string{
				"Komisi nol untuk saham dan ETF AS",
				"Riset dan alat analisis gratis untuk nasabah",
				"Diawasi regulator AS dan aset diasuransikan SIPC",
			},
			Cons: []string{
				"Biaya konversi mata uang untuk dana dari luar AS",
				"Biaya penarikan dana internasional cukup mahal",
				"Pembukaan akun untuk non-penduduk AS terbatas dan berbelit",
			},
		},
		{
			Name:        "Webull",
			Category:    "saham",
			Type:        "Luar Negeri",
			LogoDomain:  "webull.com",
			Description: "Broker digital asal Amerika Serikat dengan aplikasi ringan, grafik lengkap, dan komisi nol untuk saham AS.",
			Pros: []string{
				"Komisi nol untuk saham dan ETF AS",
				"Grafik dan alat analisis teknikal lengkap",
				"Bisa membeli saham pecahan (fractional)",
			},
			Cons: []string{
				"Biaya langganan untuk data pasar real-time",
				"Deposit internasional butuh konversi ke dolar",
				"Ketersediaan akun untuk penduduk Indonesia terbatas",
			},
		},
		{
			Name:        "eToro",
			Category:    "saham",
			Type:        "Luar Negeri",
			LogoDomain:  "etoro.com",
			Description: "Platform sosial investasi internasional yang memungkinkan pengguna menyalin portofolio trader lain.",
			Pros: []string{
				"Fitur copy trading mudah diikuti pemula",
				"Komisi nol untuk pembelian saham",
				"Bisa mulai dari nominal kecil",
			},
			Cons: []string{
				"Biaya penarikan dana cukup mahal, sekitar 5 dolar per transaksi",
				"Fee tersembunyi pada spread harga instrumen",
				"Jenis izin berbeda antar negara dan perlu diperiksa",
			},
		},
		{
			Name:        "Moomoo",
			Category:    "saham",
			Type:        "Luar Negeri",
			LogoDomain:  "moomoo.com",
			Description: "Aplikasi dari grup Futu yang menyatukan saham Amerika, Hong Kong, dan Singapura dengan biaya transaksi rendah.",
			Pros: []string{
				"Fee transaksi rendah dan tanpa komisi untuk saham AS",
				"Data pasar dan riset lengkap di dalam aplikasi",
				"Ada saham pecahan dan reksa dana",
			},
			Cons: []string{
				"Langganan data pasar real-time untuk beberapa bursa",
				"Biaya konversi mata uang untuk transaksi lintas pasar",
				"Belum berizin OJK, perlindungan regulator lokal tidak berlaku",
			},
		},
		{
			Name:        "Tiger Brokers",
			Category:    "saham",
			Type:        "Luar Negeri",
			LogoDomain:  "itiger.com",
			Description: "Broker online lintas negara dengan akses saham Amerika, Hong Kong, Singapura, dan Tiongkok dari satu akun.",
			Pros: []string{
				"Fee transaksi rendah dan kompetitif antar pasar",
				"Satu akun untuk beberapa bursa Asia dan Amerika",
				"Ada fitur riset dan komunitas investor",
			},
			Cons: []string{
				"Biaya konversi mata uang menambah biaya transaksi",
				"Biaya data pasar real-time berlaku untuk sebagian bursa",
				"Tidak diawasi OJK, perlindungan lokal tidak ada",
			},
		},
		{
			Name:        "Pluang",
			Category:    "komoditas",
			Type:        "Lokal",
			LogoDomain:  "pluang.com",
			Description: "Super-app investasi yang menyatukan emas digital, reksa dana, saham AS, hingga kripto dalam satu aplikasi.",
			Pros: []string{
				"Emas digital bisa dibeli mulai dari 0,01 gram",
				"Tidak ada biaya administrasi bulanan",
				"Emas disimpan di kustodian yang diawasi",
			},
			Cons: []string{
				"Fee tersembunyi pada selisih harga (spread) emas",
				"Sebagian fitur terbatas pada akun tertentu",
				"Terlalu banyak produk bisa membingungkan pemula",
			},
		},
		{
			Name:        "Treasury",
			Category:    "komoditas",
			Type:        "Lokal",
			LogoDomain:  "treasury.id",
			Description: "Platform emas digital berizin Bappebti dengan kepemilikan emas fisik nyata yang tercatat atas nama pengguna.",
			Pros: []string{
				"Pembelian emas bisa dimulai dari Rp5.000 tanpa biaya admin",
				"Emas fisik tersimpan di kustodian, terpisah dari aset perusahaan",
				"Emas bisa dicetak menjadi batangan fisik",
			},
			Cons: []string{
				"Biaya cetak emas fisik dan harga buyback memotong keuntungan",
				"Hanya menyediakan produk emas",
				"Belum ada pilihan kelas aset lain",
			},
		},
		{
			Name:        "Antam (Logam Mulia)",
			Category:    "komoditas",
			Type:        "Lokal",
			LogoDomain:  "logammulia.com",
			Description: "Situs resmi Logam Mulia milik Antam untuk membeli emas batangan bersertifikat langsung dari perusahaan.",
			Pros: []string{
				"Emas batangan bersertifikat resmi dari Antam",
				"Harga mengikuti acuan harga Logam Mulia harian",
				"Pembelian minimum satu gram tanpa biaya admin bulanan",
			},
			Cons: []string{
				"Biaya cetak dan pengiriman emas fisik relatif mahal",
				"Harga jual kembali (buyback) lebih rendah dari harga beli",
				"Belum ada fitur tabungan atau pembelian sebagian",
			},
		},
		{
			Name:        "Pegadaian",
			Category:    "komoditas",
			Type:        "Lokal",
			LogoDomain:  "pegadaian.co.id",
			Description: "Layanan Tabungan Emas dari Pegadaian, BUMN dengan jaringan kantor cabang di seluruh Indonesia.",
			Pros: []string{
				"Bisa menabung emas mulai dari 0,01 gram",
				"Biaya administrasi tahunan tetap, bukan per transaksi",
				"Jaringan kantor cabang luas di seluruh Indonesia",
			},
			Cons: []string{
				"Biaya administrasi tahunan tetap membebani saldo kecil",
				"Harga beli kembali mengikuti harga buyback Pegadaian",
				"Aplikasinya belum sefleksibel platform digital murni",
			},
		},
		{
			Name:        "Indogold",
			Category:    "komoditas",
			Type:        "Lokal",
			LogoDomain:  "indogold.id",
			Description: "Platform jual-beli emas digital lokal dengan layanan cicilan emas dan cetak fisik kapan saja.",
			Pros: []string{
				"Bisa mulai dari Rp10.000 tanpa biaya buka akun",
				"Ada program cicilan emas dengan tenor tetap",
				"Emas bisa dicetak menjadi batangan fisik",
			},
			Cons: []string{
				"Biaya cetak dan kirim emas fisik berlaku",
				"Fee tersembunyi pada selisih harga jual dan buyback",
				"Pilihan aset hanya terbatas pada emas",
			},
		},
	}
}
