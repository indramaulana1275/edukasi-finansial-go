package seeders

import (
	"edukasi-finansial-goedukasi-finansial-go/app/facades"
	"edukasi-finansial-goedukasi-finansial-go/app/models"
)

type EducationModuleSeeder struct{}

// Signature The name and signature of the seeder.
func (s *EducationModuleSeeder) Signature() string {
	return "EducationModuleSeeder"
}

// Run mengisi modul edukasi saat tabelnya masih kosong. Setelah admin mulai
// mengelola modul, seeder tidak lagi menimpa judul, deskripsi, ikon, atau slide
// yang sudah diedit; satu-satunya perbaikan otomatis adalah mengisi ikon kosong
// pada instalasi lama. Modul yang dihapus admin juga tidak muncul kembali.
func (s *EducationModuleSeeder) Run() error {
	existing := make([]models.EducationModule, 0)
	if err := facades.Orm().Query().Get(&existing); err != nil {
		return err
	}

	// Instalasi baru: isi seluruh materi awal.
	if len(existing) == 0 {
		seeds := educationModuleSeeds()
		for index := range seeds {
			seeds[index].Icon = educationModuleIcon(seeds[index].Title)
		}

		return facades.Orm().Query().Create(&seeds)
	}

	// Instalasi lama: hanya lengkapi ikon yang belum ada. Konten yang sudah
	// diubah lewat Panel Developer dibiarkan apa adanya.
	byTitle := make(map[string]models.EducationModule, len(existing))
	for _, module := range existing {
		byTitle[module.Title] = module
	}

	for _, seed := range educationModuleSeeds() {
		old, found := byTitle[seed.Title]
		if !found || old.Icon != "" {
			continue
		}

		old.Icon = educationModuleIcon(seed.Title)
		if err := facades.Orm().Query().Save(&old); err != nil {
			return err
		}
	}

	// Dua modul kategori baru disisipkan pada instalasi lama bila belum ada.
	// Modul lain yang sudah diedit atau dihapus admin tidak disentuh.
	for _, title := range []string{"Mengenal SBSN/Sukuk", "Investasi Logam Mulia"} {
		if _, found := byTitle[title]; found {
			continue
		}

		for _, seed := range educationModuleSeeds() {
			if seed.Title != title {
				continue
			}

			seed.Icon = educationModuleIcon(seed.Title)
			if err := facades.Orm().Query().Create(&seed); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// slidesJSON menyusun kolom content dari daftar slide. Dipakai seeder, bukan
// hardcode string JSON, supaya isi materinya tetap enak dibaca di kode.
func slidesJSON(slides ...models.EducationSlide) string {
	return models.SlidesToJSON(slides)
}

// educationModuleIcon menjaga ikon bawaan tetap sama dengan judul modulnya.
// Admin tetap bisa mengganti ikon lewat Panel Developer; peta ini hanya dipakai
// seeder supaya instalasi baru langsung punya tampilan yang rapi.
func educationModuleIcon(title string) string {
	icons := map[string]string{
		"Dasar Investasi":       "🌱",
		"Apa itu DCA?":          "📊",
		"Profil Risiko":         "🛡️",
		"Diversifikasi":         "🧺",
		"Disiplin dan Emosi":    "🧘",
		"Mengenal Saham":        "📈",
		"Reksadana vs Saham":    "⚖️",
		"Cara Membaca Grafik":   "📉",
		"Pajak Investasi":       "🧾",
		"Psikologi Investasi":   "🧠",
		"Mengenal SBSN/Sukuk":   "📜",
		"Investasi Logam Mulia": "🥇",
	}

	return icons[title]
}

// educationModuleSeeds berisi seluruh materi: 10 modul, masing-masing 10 slide
// berurutan dari pengenalan sampai langkah praktis.
func educationModuleSeeds() []models.EducationModule {
	return []models.EducationModule{
		{
			Title:       "Dasar Investasi",
			Description: "Langkah pertama sebelum menaruh uang di aset apa pun: pahami tujuan, inflasi, dan risikonya.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Apa itu investasi?",
					Body:  "Investasi adalah menempatkan dana pada aset produktif dengan harapan nilainya tumbuh melebihi inflasi. Berbeda dari menabung yang fokus menjaga nominal, investasi menerima naik-turun nilai sebagai konsekuensi dari potensi imbal hasil.",
				},
				models.EducationSlide{
					Title: "Inflasi menggerus daya beli",
					Body:  "Harga barang naik setiap tahun, sehingga uang yang dibiarkan diam kehilangan daya belinya. Karena itu imbal hasil investasi perlu dibandingkan dengan inflasi, bukan hanya dengan angka nol.",
				},
				models.EducationSlide{
					Title: "Risiko dan imbal hasil sejalan",
					Body:  "Semakin besar potensi keuntungan, semakin besar pula kemungkinan kerugiannya. Tidak ada aset berimbal hasil tinggi tanpa risiko; yang bisa kamu atur hanyalah seberapa besar risiko yang siap kamu tanggung.",
				},
				models.EducationSlide{
					Title: "Bunga majemuk bekerja untukmu",
					Body:  "Imbal hasil yang diinvestasikan ulang ikut menghasilkan imbal hasil pada periode berikutnya. Efeknya terasa kecil di tahun pertama, tetapi dalam sepuluh atau dua puluh tahun pertumbuhannya melengkung tajam. Waktu di pasar biasanya lebih menentukan daripada besarnya modal awal.",
				},
				models.EducationSlide{
					Title: "Sejarah singkat: dari tanah ke bursa",
					Body:  "Manusia sudah berbagi hasil usaha sejak zaman perdagangan rempah: satu pihak menyediakan modal, pihak lain menjalankan usaha, untung dan rugi dibagi. Gagasan itulah yang sekarang menjelma jadi saham, obligasi, dan reksadana di bursa modern.",
				},
				models.EducationSlide{
					Title: "Dana darurat dulu, investasi kemudian",
					Body:  "Siapkan dana darurat 3-6 bulan pengeluaran di rekening yang mudah dicairkan sebelum berinvestasi. Tanpa bantalan itu, kebutuhan mendadak memaksa kamu menjual aset di waktu yang buruk dan mengubah kerugian sementara menjadi kerugian permanen.",
				},
				models.EducationSlide{
					Title: "Peta instrumen",
					Body:  "Tabungan dan deposito mengutamakan keamanan, obligasi memberi arus bunga, saham mengejar pertumbuhan, emas berperan sebagai lindung nilai, sedangkan reksadana adalah wadah yang mengelola kumpulan aset itu untukmu. Kenali dulu satu per satu sebelum menaruh dana.",
				},
				models.EducationSlide{
					Title: "Jangka waktu menentukan pilihan",
					Body:  "Uang yang dibutuhkan dalam satu-dua tahun sebaiknya tidak ditaruh di aset yang berfluktuasi, karena kamu belum tentu punya waktu menunggu pemulihan harga. Semakin panjang jangka waktunya, semakin besar porsi aset tumbuh yang bisa kamu pertimbangkan.",
				},
				models.EducationSlide{
					Title: "Dua cara mendapat keuntungan",
					Body:  "Keuntungan bisa datang dari selisih harga jual dan beli (capital gain) atau dari pembagian hasil seperti bunga dan dividen (income). Ada investor yang mengandalkan pertumbuhan, ada yang mengandalkan arus kas; keduanya sah selama cocok dengan tujuanmu.",
				},
				models.EducationSlide{
					Title: "Langkah pertama yang praktis",
					Body:  "Tulis tujuan dan jangka waktunya, hitung dana yang benar-benar bisa kamu relakan, pilih satu instrumen yang paling kamu pahami, lalu mulai dengan nominal kecil secara rutin. Catat alasan tiap keputusan dan tinjau ulang setiap enam bulan, bukan setiap hari.",
				},
			),
		},
		{
			Title:       "Apa itu DCA?",
			Description: "Strategi membeli bertahap dengan nominal tetap, dan kapan cara ini lebih cocok dipakai.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Definisi singkat",
					Body:  "Dollar Cost Averaging (DCA) adalah membeli aset dengan nominal tetap secara berkala, misalnya setiap dua minggu, tanpa peduli harga sedang naik atau turun. Keputusannya diambil sekali di awal, lalu dijalankan berulang.",
				},
				models.EducationSlide{
					Title: "Kenapa membantu?",
					Body:  "Karena nominalnya tetap, kamu otomatis membeli lebih banyak unit saat harga murah dan lebih sedikit saat harga mahal. Harga rata-rata beli jadi lebih halus dan kamu tidak perlu menebak waktu terbaik untuk masuk.",
				},
				models.EducationSlide{
					Title: "Contoh sederhana",
					Body:  "Kamu menyisihkan Rp500.000 dua kali sebulan. Saat harga turun, uang yang sama mendapat unit lebih banyak; saat harga naik, unitnya berkurang. Setelah setahun, harga rata-rata belimu berada di tengah-tengah rentang harga yang dilalui pasar.",
				},
				models.EducationSlide{
					Title: "Kapan kurang cocok",
					Body:  "DCA mengasumsikan asetnya layak dipegang jangka panjang. Untuk aset yang nilainya cenderung turun permanen atau perusahaan yang usahanya memburuk, membeli rutin hanya memperbesar kerugian.",
				},
				models.EducationSlide{
					Title: "DCA versus beli sekaligus",
					Body:  "Kalau kamu punya dana besar dan pasar sedang tidak mahal, membeli sekaligus secara statistik sering lebih menguntungkan karena uangnya langsung bekerja. DCA lebih unggul untuk dana yang datang bertahap seperti gaji atau beasiswa, dan untuk menjaga emosi tetap tenang.",
				},
				models.EducationSlide{
					Title: "Cara menghitung harga rata-rata",
					Body:  "Jumlahkan seluruh rupiah yang kamu keluarkan, lalu bagi dengan total unit yang kamu kumpulkan. Angka itu yang menentukan untung-rugimu, bukan harga pada pembelian terakhir. Rata-rata inilah yang membuat fluktuasi harian tidak terlalu penting.",
				},
				models.EducationSlide{
					Title: "Pilih frekuensi yang realistis",
					Body:  "Mingguan, bulanan, atau setiap kali gajian sama-sama sah. Yang penting bukan frekuensinya, melainkan apakah jadwal itu benar-benar bisa kamu penuhi selama bertahun-tahun. Jadwal yang terlewat karena tidak realistis menghapus manfaat dari strategi ini.",
				},
				models.EducationSlide{
					Title: "Nominal tetap itu relatif",
					Body:  "Nominalnya cukup tetap dalam satu periode, misalnya setahun, lalu ditinjau saat penghasilan berubah. Menaikkan nominal seiring kenaikan gaji justru mempercepat hasilnya; yang dihindari adalah mengubah nominal karena panik melihat harga.",
				},
				models.EducationSlide{
					Title: "DCA bukan jaminan untung",
					Body:  "DCA hanya meratakan harga beli, bukan menghapus risiko. Kalau setelah lima tahun nilainya tetap di bawah, masalahnya ada pada pilihan aset dan biayanya, bukan pada metode pembeliannya.",
				},
				models.EducationSlide{
					Title: "Jadikan kebiasaan otomatis",
					Body:  "Pakai fitur autodebet atau jadwal transfer supaya pembelian berjalan tanpa perlu diputuskan ulang setiap bulan. Keputusan yang berulang setiap hari hanya menambah peluang emosi ikut campur.",
				},
			),
		},
		{
			Title:       "Profil Risiko",
			Description: "Seberapa besar naik-turun nilai yang masih nyaman kamu hadapi, dan apa pengaruhnya pada porsi aset.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Kenapa perlu diukur",
					Body:  "Profil risiko menggambarkan seberapa besar fluktuasi yang masih nyaman kamu hadapi. Dengan mengetahuinya, komposisi aset bisa disesuaikan dengan toleransi, bukan sekadar meniru pilihan orang lain.",
				},
				models.EducationSlide{
					Title: "Tiga kelompok",
					Body:  "Konservatif fokus menjaga nilai dengan porsi kecil aset volatil. Moderat mencari keseimbangan. Agresif menyiapkan porsi besar di aset tumbuh dan siap menerima penurunan sementara di atas 30%.",
				},
				models.EducationSlide{
					Title: "Jangka waktu juga menentukan",
					Body:  "Tujuan kurang dari dua tahun umumnya lebih cocok pada instrumen berisiko rendah, sedangkan tujuan panjang baru cocok memakai aset yang berfluktuasi. Profil risiko dan jangka waktu selalu dibaca bersama-sama.",
				},
				models.EducationSlide{
					Title: "Bukan label permanen",
					Body:  "Profil bisa berubah saat penghasilan, tujuan, atau toleransi risikomu berubah. Ulangi kuisnya dan sesuaikan porsi aset, bukan sebaliknya.",
				},
				models.EducationSlide{
					Title: "Toleransi versus kemampuan",
					Body:  "Ada dua hal berbeda yang sering dicampur. Kemampuan menanggung risiko berbicara tentang kondisi keuangan: apakah kamu masih bisa hidup kalau dana itu turun setengah. Toleransi berbicara tentang perutmu: apakah kamu masih bisa tidur nyenyak. Pilihan akhir harus lolos keduanya.",
				},
				models.EducationSlide{
					Title: "Kenapa pertanyaannya diulang",
					Body:  "Kuesioner yang baik menanyakan hal serupa dengan cara berbeda untuk menguji konsistensi jawaban. Kalau dua jawaban bertentangan, biasanya yang muncul adalah niat baik di atas kertas dan reaksi sebenarnya saat harga jatuh.",
				},
				models.EducationSlide{
					Title: "Uji dengan skenario penurunan",
					Body:  "Bayangkan portofolio Rp10.000.000 turun menjadi Rp7.000.000 dalam enam bulan. Kalau reaksimu adalah menambah sedikit dan menunggu, profilmu cenderung agresif. Kalau kamu langsung ingin menjual semuanya, porsi aset berisikomu perlu dikurangi sekarang, bukan nanti saat kejadian.",
				},
				models.EducationSlide{
					Title: "Porsi aset per profil",
					Body:  "Sebagai gambaran kasar, konservatif menaruh mayoritas dana di pasar uang dan obligasi dengan porsi kecil di saham, moderat membagi cukup berimbang, sedangkan agresif memberi porsi dominan pada saham dan aset tumbuh. Angka pastinya bergantung pada tujuan, penghasilan, dan usia.",
				},
				models.EducationSlide{
					Title: "Risiko yang bisa dihindari",
					Body:  "Risiko pasar karena harga naik-turun harus diterima sebagai harga dari potensi imbal hasil. Risiko lain seperti menaruh semua dana di satu aset, memakai uang pinjaman, atau membeli tanpa memahami produknya bisa kamu hindari sepenuhnya.",
				},
				models.EducationSlide{
					Title: "Setelah tahu profilmu",
					Body:  "Terjemahkan profilnya menjadi target alokasi tertulis, pilih instrumen yang sesuai, lalu tinjau ulang setiap enam bulan atau saat tujuanmu berubah. Profil yang tidak pernah dipakai untuk menyusun alokasi hanya jadi angka tanpa arti.",
				},
			),
		},
		{
			Title:       "Diversifikasi",
			Description: "Menyebar dana ke beberapa aset agar satu penurunan tidak menjatuhkan seluruh portofolio.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Jangan taruh telur di satu keranjang",
					Body:  "Diversifikasi berarti menyebar dana ke beberapa aset yang pergerakannya tidak selalu bersamaan, sehingga satu aset yang sedang jatuh tidak menjatuhkan seluruh portofolio.",
				},
				models.EducationSlide{
					Title: "Kelas aset yang berbeda",
					Body:  "Kas dan deposito menekankan keamanan, obligasi pada arus kas, saham pada pertumbuhan, dan komoditas seperti emas pada lindung nilai. Masing-masing punya peran, dan perannya baru terasa saat yang lain sedang lesu.",
				},
				models.EducationSlide{
					Title: "Porsi yang masuk akal",
					Body:  "Porsi tepat bergantung pada profil risiko dan tujuan. Kuncinya adalah menetapkannya di awal dan menyesuaikan secara berkala, bukan mengejar aset yang sedang naik paling tinggi bulan ini.",
				},
				models.EducationSlide{
					Title: "Korelasi adalah kuncinya",
					Body:  "Diversifikasi baru bekerja kalau aset-asetnya tidak bergerak searah. Memiliki lima saham dari sektor yang sama atau sepuluh kripto berbeda tidak menyebar risiko sebanyak yang terlihat. Yang dicari adalah aset yang saat satu turun, yang lain tidak ikut turun.",
				},
				models.EducationSlide{
					Title: "Risiko pasar tetap ada",
					Body:  "Saat krisis besar, hampir semua aset bisa turun bersamaan. Diversifikasi memperkecil kerusakan pada satu aset, bukan menghapus kemungkinan seluruh portofolio merah. Jangan jadikan diversifikasi alasan untuk merasa kebal.",
				},
				models.EducationSlide{
					Title: "Terlalu banyak juga tidak baik",
					Body:  "Memegang puluhan instrumen kecil-kecil membuat pantauan dan rebalancing merepotkan tanpa menambah manfaat berarti. Sepuluh sampai lima belas posisi di empat kelas aset biasanya sudah cukup menyebar untuk portofolio pribadi.",
				},
				models.EducationSlide{
					Title: "Rebalancing menjaga porsi",
					Body:  "Setelah beberapa bulan, aset yang menang cenderung membengkak dan membuat portofolio jadi lebih berisiko dari rencana. Rebalancing berarti mengembalikan porsi ke target awal, biasanya dengan menjual sedikit yang naik dan menambah yang tertinggal.",
				},
				models.EducationSlide{
					Title: "Diversifikasi lintas waktu",
					Body:  "Menyebar pembelian dari waktu ke waktu lewat DCA adalah bentuk diversifikasi lain: kamu tidak bertaruh pada satu harga masuk. Menggabungkan keduanya membuat hasil akhir tidak bergantung pada satu aset maupun satu momen.",
				},
				models.EducationSlide{
					Title: "Contoh portofolio mahasiswa",
					Body:  "Dana darurat di tabungan, inti portofolio di reksadana pasar uang dan obligasi, lalu porsi kecil di reksadana saham atau satu-dua saham pilihan. Nominalnya boleh kecil; yang latihan sejak sekarang adalah struktur dan kebiasaannya.",
				},
				models.EducationSlide{
					Title: "Kesalahan umum",
					Body:  "Menyebut diri terdiversifikasi padahal semua dananya ada di satu reksadana saham yang isinya mirip, membeli aset kedua hanya karena sedang tren, dan lupa menghitung biaya. Periksa isi produknya, bukan hanya namanya.",
				},
			),
		},
		{
			Title:       "Disiplin dan Emosi",
			Description: "Menjaga rencana tetap berjalan saat pasar bergerak liar, dan kenapa itu lebih penting daripada analisis.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Musuh terbesar adalah diri sendiri",
					Body:  "Banyak kerugian lahir dari panik saat harga turun atau tergoda ikut-ikutan saat harga memuncak, bukan dari analisis yang salah.",
				},
				models.EducationSlide{
					Title: "Aturan yang dibuat di awal",
					Body:  "Tetapkan nominal, frekuensi, dan target alokasi saat kepala dingin. Aturan yang sudah tertulis jauh lebih mudah dipatuhi ketika pasar bergerak liar.",
				},
				models.EducationSlide{
					Title: "Evaluasi, jangan menebak",
					Body:  "Periksa portofolio secara berkala dan sesuaikan ke target alokasi. Mengevaluasi rencana berbeda dari mencoba menebak arah pasar jangka pendek.",
				},
				models.EducationSlide{
					Title: "Tulis rencana investasimu",
					Body:  "Cukup satu halaman: tujuan, jangka waktu, target alokasi, nominal rutin, dan alasan memilih tiap instrumen. Rencana tertulis membuat keputusan berikutnya menjadi pemeriksaan terhadap rencana, bukan negosiasi ulang dengan emosi.",
				},
				models.EducationSlide{
					Title: "Kenapa kita panik saat merah",
					Body:  "Penurunan harga terasa seperti ancaman, dan tubuh bereaksi seperti menghadapi bahaya nyata: jantung berdebar, ingin segera bertindak. Kesadaran bahwa reaksi itu wajar membuat kamu bisa menunda keputusan sehari sebelum menekan tombol jual.",
				},
				models.EducationSlide{
					Title: "Godaan mengejar kinerja masa lalu",
					Body:  "Produk yang tahun lalu naik paling tinggi paling sering muncul di iklan, padahal kinerja masa lalu bukan jaminan. Data historis dipakai untuk memahami risiko, bukan untuk memprediksi urutan pemenang tahun depan.",
				},
				models.EducationSlide{
					Title: "Daftar periksa sebelum transaksi",
					Body:  "Sebelum membeli atau menjual, tanyakan: apa alasannya, apakah sesuai rencana, apakah ini reaksi terhadap harga hari ini, dan bagaimana pengaruhnya ke alokasi. Empat pertanyaan itu menyaring sebagian besar keputusan impulsif.",
				},
				models.EducationSlide{
					Title: "Jangan periksa harga tiap menit",
					Body:  "Semakin sering kamu melihat harga, semakin banyak keputusan yang terasa mendesak padahal tidak. Untuk strategi jangka panjang, cek portofolio saat jadwal rutin tiba atau saat rebalancing, bukan setiap kali pasar bergerak.",
				},
				models.EducationSlide{
					Title: "Pengaruh lingkungan",
					Body:  "Obrolan grup dan berita sensasional bisa membuat rencana yang sudah baik terasa salah. Boleh mendengar, tetapi keputusan tetap berpijak pada tujuanmu sendiri. Kalau sebuah ide tidak bisa dijelaskan dalam satu paragraf, biasanya itu bukan rencana.",
				},
				models.EducationSlide{
					Title: "Jurnal keputusan",
					Body:  "Catat tanggal, alasan, dan ekspektasimu setiap kali bertransaksi. Beberapa bulan kemudian kamu bisa menilai apakah keputusanmu berdasarkan data atau suasana hati. Jurnal yang jujur adalah guru investasi paling murah yang bisa kamu miliki.",
				},
			),
		},
		{
			Title:       "Mengenal Saham",
			Description: "Apa sebenarnya yang kamu beli saat membeli saham, dari mana keuntungannya, dan apa risikonya.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Saham adalah kepemilikan",
					Body:  "Membeli saham berarti memiliki sebagian kecil dari perusahaan tersebut. Kamu ikut menikmati hasilnya saat perusahaan untung, dan ikut terdampak saat usahanya memburuk.",
				},
				models.EducationSlide{
					Title: "Dua sumber keuntungan",
					Body:  "Pertama, capital gain: harga jual lebih tinggi dari harga beli. Kedua, dividen: sebagian laba perusahaan yang dibagikan ke pemegang saham. Saham yang tidak membagikan dividen menahan labanya untuk tumbuh.",
				},
				models.EducationSlide{
					Title: "Kenapa harganya bergerak",
					Body:  "Harga di bursa ditentukan oleh tawar-menawar pembeli dan penjual. Laporan keuangan, prospek industri, suku bunga, dan sentimen pasar semuanya masuk ke dalam harga, sehingga fluktuasi harian itu normal.",
				},
				models.EducationSlide{
					Title: "Mulai dari yang kamu pahami",
					Body:  "Transaksi dihitung dalam lot, di mana satu lot sama dengan seratus lembar. Mulailah dari perusahaan yang model bisnisnya kamu mengerti, pahami laporan keuangannya, dan jangan memakai dana darurat untuk membeli saham.",
				},
				models.EducationSlide{
					Title: "Bursa dan jam perdagangan",
					Body:  "Saham diperdagangkan di bursa efek pada hari kerja dalam dua sesi, pagi dan siang. Harga bergerak naik turun dalam batas auto rejection sebagai pengaman, dan setiap transaksi dikenai biaya jual beli yang dipotong oleh sekuritas.",
				},
				models.EducationSlide{
					Title: "Indeks sebagai termometer pasar",
					Body:  "Indeks menggabungkan banyak saham menjadi satu angka, misalnya IHSG untuk seluruh pasar atau indeks sektoral untuk industri tertentu. Indeks memberi gambaran arah pasar secara umum, sementara performa sahammu bisa berbeda jauh dari indeks itu.",
				},
				models.EducationSlide{
					Title: "Membaca laporan keuangan singkat",
					Body:  "Tiga hal pertama yang perlu kamu lihat adalah pertumbuhan pendapatan, laba bersih, dan utang. Dari situ kamu bisa menghitung laba per saham serta rasio harga terhadap laba untuk menilai apakah harganya mahal atau murah dibandingkan kinerjanya.",
				},
				models.EducationSlide{
					Title: "Dividen dan jadwalnya",
					Body:  "Perusahaan yang membagikan dividen biasanya mengumumkan jadwalnya setelah rapat pemegang saham. Imbal hasil dividen dihitung dari dividen tahunan dibagi harga saham saat ini. Dividen besar tidak selalu bagus kalau perusahaan jadi kekurangan dana untuk tumbuh.",
				},
				models.EducationSlide{
					Title: "Aksi korporasi",
					Body:  "Selain dividen, ada aksi seperti stock split yang memecah satu saham menjadi beberapa, right issue yang menawarkan saham baru kepada pemegang saham lama, dan buyback oleh perusahaan. Semua itu mengubah jumlah lembar, jadi harga sebelum dan sesudahnya tidak bisa dibandingkan langsung.",
				},
				models.EducationSlide{
					Title: "Risiko saham dan mengelolanya",
					Body:  "Saham bisa turun jauh saat kinerja perusahaan mengecewakan atau pasar panik, dan uangmu tidak dijamin. Kelola dengan porsi yang sesuai profil, membeli perusahaan sehat secara bertahap, menyebar ke beberapa saham atau reksadana, dan berpikir dalam hitungan tahun.",
				},
			),
		},
		{
			Title:       "Reksadana vs Saham",
			Description: "Membandingkan dua jalur investasi populer: menyerahkan pengelolaan ke manajer investasi atau membeli sendiri.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Reksadana itu apa",
					Body:  "Reksadana adalah wadah yang mengumpulkan dana banyak investor lalu dikelola oleh manajer investasi. Kamu tidak membeli saham secara langsung, melainkan membeli unit penyertaan dari portofolio yang dikelola.",
				},
				models.EducationSlide{
					Title: "Jenisnya bertingkat risiko",
					Body:  "Reksadana pasar uang paling stabil, pendapatan tetap pada obligasi, campuran menggabungkan keduanya, dan reksadana saham paling berfluktuasi. Semakin ke kanan, potensi imbal hasil dan naik-turunnya sama-sama bertambah.",
				},
				models.EducationSlide{
					Title: "Kontrol atau kepraktisan",
					Body:  "Saham memberi kontrol penuh atas pilihan dan waktu beli, tapi menuntut waktu untuk riset. Reksadana memindahkan pekerjaan itu ke manajer investasi dengan imbalan biaya pengelolaan, dan nilai investasinya dihitung per unit.",
				},
				models.EducationSlide{
					Title: "Tidak harus memilih satu",
					Body:  "Banyak pemula memakai reksadana untuk inti portofolio dan saham untuk porsi kecil yang mau mereka pelajari sendiri. Yang penting bukan menang-menangan, tapi porsi yang sesuai dengan profil risikomu.",
				},
				models.EducationSlide{
					Title: "Biaya yang perlu dibaca",
					Body:  "Ada biaya pengelolaan tahunan yang dipotong dari nilai aktiva, biaya pembelian, biaya penjualan, dan biaya pengalihan antar reksadana. Terlihat kecil, tapi biaya tahunan satu persen yang berjalan dua puluh tahun memakan bagian hasil yang cukup besar. Selalu baca prospektusnya.",
				},
				models.EducationSlide{
					Title: "NAB dan nilai investasimu",
					Body:  "Nilai Aktiva Bersih per unit adalah harga satu unit reksadana dan berubah setiap hari setelah pasar tutup. Nilai investasimu sama dengan jumlah unit yang kamu miliki dikali NAB saat itu, bukan nominal yang kamu setorkan.",
				},
				models.EducationSlide{
					Title: "Likuiditas dan pencairan",
					Body:  "Saham bisa dijual saat bursa buka, sedangkan reksadana dicairkan mengikuti jadwal penerbitnya: pasar uang umumnya satu-dua hari kerja, pendapatan tetap bisa lebih lama, dan reksadana saham mengikuti siklus perdagangan. Rencanakan kebutuhan dana darurat di luar instrumen yang lambat cair.",
				},
				models.EducationSlide{
					Title: "Cara membeli yang sah",
					Body:  "Reksadana dibeli lewat agen penjual yang terdaftar di OJK, bisa berupa sekuritas, bank, manajer investasi, atau platform digital reksadana. Pastikan produk dan penjualnya terdaftar agar hakmu sebagai investor terlindungi.",
				},
				models.EducationSlide{
					Title: "Kesalahan umum investor reksadana",
					Body:  "Membeli karena peringkat kinerja satu tahun, berpindah jenis setiap kali nilainya turun, dan tidak tahu isi portofolionya. Reksadana mempermudah, tetapi tetap butuh kesabaran yang sama seperti memegang saham langsung.",
				},
				models.EducationSlide{
					Title: "Mana yang cocok untukmu",
					Body:  "Kalau waktumu terbatas, modalnya kecil, dan kamu ingin langsung terdiversifikasi, reksadana lebih praktis. Kalau kamu menikmati riset, sanggup menanggung fluktuasi satu saham, dan ingin memilih sendiri, saham bisa jadi sarana belajar terbaik saat dimulai dengan porsi kecil.",
				},
			),
		},
		{
			Title:       "Cara Membaca Grafik",
			Description: "Membaca sumbu, batang harga, dan volume tanpa menebak arah pasar dari satu pola.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Dua sumbu, satu cerita",
					Body:  "Sumbu mendatar menunjukkan waktu dan sumbu tegak menunjukkan harga. Rentang waktu yang berbeda menggambar cerita yang berbeda, jadi selalu periksa label periode sebelum menyimpulkan tren.",
				},
				models.EducationSlide{
					Title: "Candle dan garisan",
					Body:  "Satu batang candlestick merangkum harga pembukaan, penutupan, tertinggi, dan terendah pada satu periode. Grafik garis memakai satu nilai penutupan per periode sehingga lebih mudah dibaca untuk memantau tren jangka panjang.",
				},
				models.EducationSlide{
					Title: "Tren: puncak dan lembah",
					Body:  "Deretan puncak dan lembah yang menanjak menandakan tren naik, dan sebaliknya untuk tren turun. Ketika puncak dan lembah tidak lagi searah, pasar sedang beralih arah atau bergerak datar menunggu katalis baru.",
				},
				models.EducationSlide{
					Title: "Volume memberi konteks",
					Body:  "Volume besar menandakan banyak pelaku pasar sependapat dengan arah harga saat itu, sementara volume tipis menandakan pergerakan itu rapuh. Harga naik dengan volume kecil sering tidak bertahan lama.",
				},
				models.EducationSlide{
					Title: "Ganti rentang waktu",
					Body:  "Grafik harian menunjukkan gejolak jangka pendek, grafik mingguan dan bulanan memperlihatkan arah besar. Tentukan dulu horizon investasimu, lalu pilih rentang yang sesuai; melihat grafik menit padahal berinvestasi bertahun-tahun hanya menambah kebisingan.",
				},
				models.EducationSlide{
					Title: "Rata-rata bergerak",
					Body:  "Rata-rata bergerak memuluskan harga dengan menghitung nilai tengah beberapa periode terakhir. Ketika harga bertahan di atas rata-rata, trennya cenderung menguat; ketika menembus ke bawah, momentumnya sedang melemah. Ini alat bantu membaca tren, bukan sinyal pasti.",
				},
				models.EducationSlide{
					Title: "Indikator momentum",
					Body:  "Indikator momentum seperti RSI membandingkan besarnya kenaikan dan penurunan terakhir dalam skala nol sampai seratus. Nilai ekstrem memberi petunjuk bahwa harga mungkin bergerak terlalu cepat, tetapi bisa tetap ekstrem lama saat trennya kuat.",
				},
				models.EducationSlide{
					Title: "Support dan resistance",
					Body:  "Support adalah area harga yang berulang kali menahan penurunan, resistance area yang berulang kali menahan kenaikan. Area ini hanya bermakna karena banyak pelaku pasar memperhatikannya, dan akan hilang begitu benar-benar ditembus.",
				},
				models.EducationSlide{
					Title: "Grafik dan fundamental",
					Body:  "Grafik merangkum apa yang sudah terjadi dan bagaimana pasar bereaksi, sedangkan analisis fundamental menjelaskan bisnis di baliknya. Memakai keduanya membuat kamu tidak membeli saham bagus di harga yang sudah terlalu mahal, atau harga murah karena bisnisnya memang memburuk.",
				},
				models.EducationSlide{
					Title: "Kesalahan membaca grafik",
					Body:  "Mencari pola setelah kejadian dan menyimpulkan seolah bisa diramal sebelumnya, memakai terlalu banyak indikator yang saling bertentangan, serta menyimpulkan tren dari beberapa titik saja. Grafik membantu memahami konteks, bukan menebak harga besok.",
				},
			),
		},
		{
			Title:       "Pajak Investasi",
			Description: "Pajak yang muncul saat jual-beli, dari mana dihitungnya, dan kenapa itu memengaruhi imbal hasil bersih.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Imbal hasil bersih yang penting",
					Body:  "Keuntungan di layar belum memperhitungkan pajak dan biaya. Yang menambah kekayaanmu adalah imbal hasil bersih setelah semua potongan itu, jadi biasakan menghitungnya sejak awal.",
				},
				models.EducationSlide{
					Title: "Pajak dihitung dari nilai transaksi",
					Body:  "Untuk saham, penjualan dikenai pajak final atas nilai transaksi bruto, bukan hanya dari keuntungannya. Artinya transaksi rugi pun tetap dikenai pajak, yang makin menegaskan pentingnya tidak bertransaksi terlalu sering.",
				},
				models.EducationSlide{
					Title: "Kupon, dividen, dan kripto",
					Body:  "Bunga obligasi dan kupon surat berharga umumnya dikenai pajak final, sedangkan dividen tertentu bisa dikecualikan bila diinvestasikan kembali di dalam negeri. Aset kripto dipajaki saat transaksi di penyedia terdaftar.",
				},
				models.EducationSlide{
					Title: "Catat dan periksa ketentuannya",
					Body:  "Simpan bukti transaksi dan cek aturan terbaru sebelum melaporkan, karena tarif dan skemanya berbeda untuk tiap instrumen serta bisa berubah. Pajak adalah biaya investasi, bukan sesuatu yang dihindari.",
				},
				models.EducationSlide{
					Title: "Pajak transaksi saham",
					Body:  "Saat kamu menjual saham, sekuritas memotong pajak final dari nilai penjualan, ditambah biaya jual dan beli. Karena pajaknya dihitung dari nilai transaksi, keuntungan yang tipis bisa habis oleh pajak dan biaya bila kamu terlalu sering keluar-masuk.",
				},
				models.EducationSlide{
					Title: "Dividen dan syarat pengecualian",
					Body:  "Dividen yang diterima orang pribadi dalam negeri dapat dikecualikan dari pajak bila diinvestasikan kembali di Indonesia dalam jangka waktu tertentu. Bila tidak diinvestasikan kembali, dividen itu menjadi objek pajak. Karena itu pilihan menahan atau membelanjakan dividen punya konsekuensi.",
				},
				models.EducationSlide{
					Title: "Obligasi dan surat berharga",
					Body:  "Kupon obligasi maupun surat berharga negara umumnya dikenai pajak final yang langsung dipotong saat pembayaran. Imbal hasil dan keuntungan selisih harga punya perlakuan berbeda, jadi hitung hasil bersihnya, bukan hanya angka kupon yang dipromosikan.",
				},
				models.EducationSlide{
					Title: "Pajak reksadana",
					Body:  "Untuk reksadana, pajak atas transaksi di dalam portofolio ditanggung di level wadah, sehingga investor tidak menghitungnya sendiri. Namun kamu tetap perlu memperhatikan biaya pembelian dan penjualan unit, karena keduanya mengurangi hasil yang kamu terima.",
				},
				models.EducationSlide{
					Title: "Aset kripto dan digital",
					Body:  "Transaksi aset kripto di bursa atau penyedia terdaftar dikenai pungutan saat jual dan beli, jadi setiap perputaran menambah biaya. Untuk strategi jangka panjang, menahan lebih lama biasanya lebih hemat daripada sering berpindah aset.",
				},
				models.EducationSlide{
					Title: "Menghitung imbal hasil setelah pajak",
					Body:  "Kurangi seluruh keuntungan dengan pajak dan biaya jual beli, lalu bagi dengan modal awalmu. Contohnya untung Rp200.000 dari modal Rp2.000.000, dikurangi pajak dan biaya, memberi persentase bersih yang lebih kecil dari tampilan kotor. Angka bersih inilah yang dibandingkan dengan inflasi.",
				},
			),
		},
		{
			Title:       "Psikologi Investasi",
			Description: "Pola pikir yang membuat investor membeli di puncak dan menjual di dasar, serta cara menahannya.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "FOMO dan euforia",
					Body:  "Melihat tetangga cuan membuat orang membeli saat harga sudah tinggi tanpa memeriksa nilainya. Ketakutan ketinggalan sering berakhir dengan membeli di puncak dan menyesal saat harga kembali normal.",
				},
				models.EducationSlide{
					Title: "Rasa sakit kehilangan dua kali lipat",
					Body:  "Secara psikologis, kerugian terasa jauh lebih menyakitkan daripada kenikmatan untung dengan jumlah sama. Karena itu banyak orang menahan aset rugi terlalu lama dan buru-buru menjual yang sedang untung.",
				},
				models.EducationSlide{
					Title: "Merasa paling benar",
					Body:  "Kita cenderung hanya mencari berita yang mendukung pilihan sendiri dan mengabaikan yang membantahnya. Kebiasaan ini membuat kesalahan awal terasa seperti keputusan yang kuat.",
				},
				models.EducationSlide{
					Title: "Bikin aturan, bukan menebak",
					Body:  "Tulis alasan membeli, target, dan batas toleransi sebelumnya, lalu tinjau secara berkala. Jurnal sederhana membuat keputusan berikutnya berdasar data, bukan perasaan saat pasar sedang ramai.",
				},
				models.EducationSlide{
					Title: "Terpaku pada harga beli",
					Body:  "Harga saat kamu membeli tidak berarti apa-apa bagi pasar. Menahan aset hanya karena menunggu balik modal membuat keputusan ditentukan oleh angka masa lalu, padahal yang seharusnya dinilai adalah prospeknya sekarang.",
				},
				models.EducationSlide{
					Title: "Efek kawanan dan media",
					Body:  "Judul berita yang menakutkan membuat penurunan terasa lebih besar daripada angkanya. Ingat bahwa pasar bergerak setiap hari sementara tujuanmu berjalan dalam hitungan tahun; jangan biarkan satu judul mengubah rencana lima tahun.",
				},
				models.EducationSlide{
					Title: "Biaya yang sudah telanjur keluar",
					Body:  "Perusahaan yang kinerjanya memburuk sering ditahan karena sudah membeli banyak. Kenyataannya uang yang sudah keluar tidak bisa ditarik kembali; pertanyaan yang benar adalah kalau uang itu ada di tanganmu sekarang, apakah kamu masih mau membelinya.",
				},
				models.EducationSlide{
					Title: "Kepercayaan diri berlebihan",
					Body:  "Untung tiga kali beruntun di pasar yang sedang naik mudah disalahartikan sebagai keahlian, lalu porsi dibuat lebih besar dari rencana. Pasar yang sedang ramai membuat hampir semua orang terlihat pandai, jadi bandingkan hasilmu dengan indeks, bukan dengan perasaan.",
				},
				models.EducationSlide{
					Title: "Menunda karena takut salah",
					Body:  "Kebalikan dari terlalu berani adalah tidak pernah mulai karena menunggu waktu yang sempurna. Tidak ada, dan menunggu justru membuat modal kalah cepat dari inflasi. Mulai kecil, mulai sekarang, dan perbaiki sambil berjalan.",
				},
				models.EducationSlide{
					Title: "Kebiasaan investor jangka panjang",
					Body:  "Berinvestasi rutin sesuai jadwal, tidak mengubah rencana karena berita harian, menambah porsi saat rencana memang mengizinkan, membaca laporan berkala, dan mengevaluasi hasil dalam satuan tahun. Ketenangan yang terlihat itu sebenarnya adalah hasil dari kebiasaan, bukan bakat.",
				},
			),
		},
		{
			Title:       "Mengenal SBSN/Sukuk",
			Description: "Surat berharga syariah negara: cara kerja, imbal hasil, risiko, dan perannya di portofolio.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Apa itu SBSN?",
					Body:  "SBSN (Surat Berharga Syariah Negara) atau sukuk negara adalah surat berharga yang diterbitkan pemerintah berdasarkan prinsip syariah. Pemegangnya memiliki hak atas manfaat aset atau proyek yang mendasarinya, bukan bunga pinjaman.",
				},
				models.EducationSlide{
					Title: "Beda dengan obligasi konvensional",
					Body:  "Obligasi konvensional memberi bunga, sedangkan sukuk memberi imbalan berupa bagi hasil atau ujrah atas aset yang menjadi dasar penerbitan. Karena itu sukuk sering dipilih investor yang ingin menghindari bunga.",
				},
				models.EducationSlide{
					Title: "Jenis yang umum di Indonesia",
					Body:  "Ada sukuk ritel dan sukuk tabungan yang dijual langsung ke masyarakat dengan nominal kecil, serta SBSN jangka panjang untuk investor institusi. Tenornya beragam, dari beberapa tahun sampai puluhan tahun.",
				},
				models.EducationSlide{
					Title: "Imbal hasil dan jadwalnya",
					Body:  "Imbalan sukuk umumnya dibayarkan berkala, misalnya setiap bulan atau setiap enam bulan, sampai jatuh tempo. Ada yang memakai tingkat tetap dan ada yang mengambang mengikuti acuan pasar.",
				},
				models.EducationSlide{
					Title: "Risiko yang perlu dipahami",
					Body:  "Risiko gagal bayar pemerintah tergolong sangat rendah, tetapi bukan berarti nol secara mutlak. Risiko lain muncul saat kamu menjual sebelum jatuh tempo: harga di pasar sekunder bisa di bawah nilai beli bila suku bunga bergerak.",
				},
				models.EducationSlide{
					Title: "Cara membeli",
					Body:  "Sukuk ritel dibeli melalui agen distribusi resmi seperti bank, sekuritas, atau platform e-SBN pada masa penawaran. Pastikan agennya terdaftar dan pahami memorandum informasi sebelum memesan.",
				},
				models.EducationSlide{
					Title: "Likuiditas dan tenor",
					Body:  "Dana yang sudah ditempatkan sebaiknya sesuai jangka waktu yang direncanakan. Menjual sebelum jatuh tempo bisa dilakukan di pasar sekunder, tetapi harganya mengikuti pasar dan tidak selalu sama dengan nilai nominal.",
				},
				models.EducationSlide{
					Title: "Pajak dan biaya",
					Body:  "Imbalan sukuk dikenai pajak final sesuai ketentuan yang berlaku, ditambah biaya transaksi dari agen penjual. Hitung imbal hasil bersihnya, bukan hanya angka imbalan yang tertera di brosur.",
				},
				models.EducationSlide{
					Title: "Peran di portofolio",
					Body:  "Karena arus kasnya relatif teratur dan risikonya rendah, sukuk cocok menjadi penyeimbang saat pasar saham bergejolak. Porsinya disesuaikan dengan profil risiko dan kebutuhan dana jangka pendekmu.",
				},
				models.EducationSlide{
					Title: "Langkah praktis",
					Body:  "Tentukan tujuan dan tenor, bandingkan imbalan bersih antar seri, cek jadwal pembayaran, lalu beli pada masa penawaran lewat agen resmi. Simpan bukti kepemilikan dan tinjau ulang saat ada kebutuhan dana.",
				},
			),
		},
		{
			Title:       "Investasi Logam Mulia",
			Description: "Emas dan logam mulia lain sebagai lindung nilai: bentuk kepemilikan, biaya, dan porsi yang wajar.",
			Content: slidesJSON(
				models.EducationSlide{
					Title: "Apa itu logam mulia?",
					Body:  "Logam mulia adalah logam bernilai tinggi karena kelangkaan dan permintaannya, terutama emas, perak, dan platinum. Emas paling populer karena dianggap stabil dan mudah diperdagangkan.",
				},
				models.EducationSlide{
					Title: "Kenapa orang memilih emas",
					Body:  "Emas dipakai sebagai lindung nilai terhadap inflasi dan ketidakpastian. Namun emas tidak menghasilkan bunga atau dividen; keuntungannya berasal dari kenaikan harga saat dijual.",
				},
				models.EducationSlide{
					Title: "Bentuk kepemilikan",
					Body:  "Ada batangan, koin, perhiasan, rekening emas digital, dan reksadana emas. Batangan bersertifikat paling murni untuk investasi, sedangkan perhiasan membawa biaya pembuatan yang besar.",
				},
				models.EducationSlide{
					Title: "Apa yang menggerakkan harga",
					Body:  "Harga emas dipengaruhi nilai dolar, suku bunga riil, permintaan bank sentral, dan situasi geopolitik. Karena itu harganya bisa naik-turun tajam dalam jangka pendek.",
				},
				models.EducationSlide{
					Title: "Kadar dan sertifikat",
					Body:  "Kadar emas ditulis dalam karat atau persen, misalnya 24K setara 99,9 persen. Untuk batangan, pilih yang bersertifikat resmi dan simpan dokumennya agar mudah dijual kembali.",
				},
				models.EducationSlide{
					Title: "Biaya yang sering terlupa",
					Body:  "Selisih harga beli dan jual (spread), biaya cetak, biaya penyimpanan, dan asuransi mengurangi hasil. Perhiasan juga menyusut nilai saat dijual karena biaya pembuatannya tidak ikut dihitung.",
				},
				models.EducationSlide{
					Title: "Likuiditas",
					Body:  "Batangan bersertifikat dari penjual terdaftar lebih mudah dicairkan daripada perhiasan atau koin langka. Sebelum membeli, tanyakan bagaimana cara menjual kembali dan berapa potongannya.",
				},
				models.EducationSlide{
					Title: "Pajak dan pencatatan",
					Body:  "Perlakukan pajak dan biaya transaksi sebagai bagian dari perhitungan hasil. Simpan bukti pembelian dan catat harga beli supaya kamu bisa menghitung keuntungan bersih saat menjual.",
				},
				models.EducationSlide{
					Title: "Porsi yang wajar",
					Body:  "Emas biasanya dipakai sebagai pelengkap, bukan inti portofolio. Porsi kecil sekitar lima sampai sepuluh persen sudah cukup untuk menambah diversifikasi, tergantung profil risikomu.",
				},
				models.EducationSlide{
					Title: "Langkah praktis",
					Body:  "Beli di penjual terdaftar, pilih bentuk yang mudah dijual kembali, simpan sertifikat, dan jangan membeli karena panik melihat berita. Tinjau ulang porsi emas saat alokasi portofoliomu berubah.",
				},
			),
		},
	}
}
