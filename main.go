package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// Struktur untuk akun dengan informasi ID, password, saldo, status persetujuan, dan riwayat transaksi
type Account struct {
	ID           string
	Password     string
	Balance      float64
	Approved     bool
	Transactions []Transaction
}

// Struktur untuk transaksi berisi ID transaksi, ID akun, jenis transaksi, jumlah uang, tanggal, dan detail tambahan
type Transaction struct {
	ID        int
	AccountID string
	Type      string
	Amount    float64
	Date      string
	Details   string
}

// Struktur untuk pendaftaran akun, hanya berisi ID dan password
type Registration struct {
	ID       string
	Password string
}

// Struktur untuk permintaan top-up dengan ID permintaan, ID akun, jumlah, tanggal, dan status persetujuan
type TopUpRequest struct {
	ID        int
	AccountID string
	Amount    float64
	Date      string
	Approved  bool
}

// Deklarasi variabel global untuk menyimpan data akun, registrasi, permintaan top-up, dan mutex untuk sinkronisasi
var (
	accounts      = make(map[string]*Account)   // Menyimpan akun yang sudah terdaftar
	registrations = make(map[string]Registration) // Menyimpan data registrasi akun yang belum disetujui
	topUpRequests = []TopUpRequest{} // Menyimpan daftar permintaan top-up
	mu            sync.Mutex        // Mutex untuk menghindari kondisi balapan (race conditions)
	currentUser   *Account          // Menyimpan akun yang saat ini sedang login
	nextTopUpID   = 1               // ID unik untuk permintaan top-up berikutnya
)

const accountsFilePath = "accounts.json" // Lokasi file untuk menyimpan data akun

func main() {
	// Memuat akun dari file jika ada
	loadAccounts()
	defer saveAccounts() // Menyimpan akun saat program berakhir

	// Membaca input dari pengguna
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Welcome to the e-money system!")

	// Cek apakah file akun kosong atau tidak
	if isEmptyAccountsFile() {
		// Jika tidak ada akun, menawarkan untuk registrasi atau login admin
		fmt.Println("No accounts found. Please register as admin or register a new account.")
		for {
			fmt.Println("1. Admin Login")
			fmt.Println("2. Register Account")
			fmt.Println("3. Exit")

			if !scanner.Scan() {
				break
			}

			option := scanner.Text()
			switch option {
			case "1":
				if loginadm(scanner) { // Login admin
					adminMenu(scanner) // Menu admin
				}
			case "2":
				registerAccount(scanner) // Pendaftaran akun
			case "3":
				fmt.Println("Goodbye!")
				return
			default:
				fmt.Println("Invalid option. Please try again.")
			}
		}
	} else {
		// Jika ada akun, menawarkan login pengguna atau admin
		fmt.Println("Choose an option:")
		fmt.Println("1. User Login")
		fmt.Println("2. Admin Login")
		fmt.Println("3. Register Account")
		fmt.Println("4. Exit")

		for {
			if !scanner.Scan() {
				break
			}

			option := scanner.Text()
			switch option {
			case "1":
				if loginusr(scanner) { // Login pengguna
					userMenu(scanner) // Menu pengguna
				}
			case "2":
				if loginadm(scanner) { // Login admin
					adminMenu(scanner) // Menu admin
				}
			case "3":
				registerAccount(scanner) // Pendaftaran akun
			case "4":
				fmt.Println("Goodbye!")
				return
			default:
				fmt.Println("Invalid option. Please try again.")
			}
		}
	}
}

// Fungsi untuk mengecek apakah file akun kosong atau tidak
func isEmptyAccountsFile() bool {
	data, err := os.ReadFile(accountsFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println("Failed to read accounts file:", err)
		}
		return true
	}
	return len(data) == 0
}

// Fungsi untuk mendaftarkan akun baru
func registerAccount(scanner *bufio.Scanner) {
	fmt.Print("Enter account ID: ")
	if !scanner.Scan() {
		return
	}
	id := scanner.Text()

	fmt.Print("Enter password: ")
	if !scanner.Scan() {
		return
	}
	password := scanner.Text()

	mu.Lock() // Kunci mutex untuk sinkronisasi
	defer mu.Unlock()

	if _, exists := accounts[id]; exists {
		if _, exists := registrations[id]; exists {
			fmt.Println("Account already exists.")
			return
		}
	}

	registrations[id] = Registration{ID: id, Password: password}
	fmt.Println("Registration submitted for approval.")
}

// Fungsi untuk login admin
func loginadm(scanner *bufio.Scanner) bool {
	fmt.Print("Enter admin ID: ")
	if !scanner.Scan() {
		return false
	}
	id := scanner.Text()

	fmt.Print("Enter password: ")
	if !scanner.Scan() {
		return false
	}
	password := scanner.Text()

	mu.Lock() // Kunci mutex untuk sinkronisasi
	defer mu.Unlock()

	if id == "admin" && password == "admin" {
		currentUser = &Account{ID: "admin", Password: "admin"}
		return true
	}

	fmt.Println("Invalid admin credentials.")
	return false
}

// Fungsi untuk login pengguna
func loginusr(scanner *bufio.Scanner) bool {
	fmt.Print("Enter account ID: ")
	if !scanner.Scan() {
		return false
	}
	id := scanner.Text()

	fmt.Print("Enter password: ")
	if !scanner.Scan() {
		return false
	}
	password := scanner.Text()

	mu.Lock() // Kunci mutex untuk sinkronisasi
	defer mu.Unlock()

	account, exists := accounts[id]
	if !exists || account.Password != password || !account.Approved {
		fmt.Println("Invalid credentials or account not approved.")
		return false
	}

	currentUser = account
	return true
}

// Menu untuk pengguna dengan berbagai opsi seperti cek saldo, transfer uang, pembayaran, dll.
func userMenu(scanner *bufio.Scanner) {
	for {
		fmt.Println("1. Check Balance")
		fmt.Println("2. Transfer Money")
		fmt.Println("3. Make Payment")
		fmt.Println("4. Print Transaction History")
		fmt.Println("5. Top Up Balance")
		fmt.Println("6. Logout")

		if !scanner.Scan() {
			break
		}

		option := scanner.Text()
		switch option {
		case "1":
			checkBalance() // Cek saldo akun
		case "2":
			transferMoney(scanner) // Transfer uang ke akun lain
		case "3":
			makePayment(scanner) // Melakukan pembayaran
		case "4":
			printTransactionHistory() // Cetak riwayat transaksi
		case "5":
			topUpBalance(scanner) // Mengisi saldo akun
		case "6":
			currentUser = nil // Logout pengguna
			return
		default:
			fmt.Println("Invalid option. Please try again.")
		}
	}
}

// Menu untuk admin untuk mengelola pendaftaran dan permintaan top-up
func adminMenu(scanner *bufio.Scanner) {
	for {
		fmt.Println("1. Approve/Reject Registration")
		fmt.Println("2. Print Account List")
		fmt.Println("3. Approve/Reject Top Up Requests")
		fmt.Println("4. Logout")

		if !scanner.Scan() {
			break
		}

		option := scanner.Text()
		switch option {
		case "1":
			handleRegistrations(scanner) // Menangani pendaftaran akun baru
		case "2":
			printAccountList() // Cetak daftar akun
		case "3":
			handleTopUpRequests(scanner) // Menangani permintaan top-up
		case "4":
			currentUser = nil // Logout admin
			return
		default:
			fmt.Println("Invalid option. Please try again.")
		}
	}
}

// Fungsi untuk menangani persetujuan pendaftaran akun baru oleh admin
func handleRegistrations(scanner *bufio.Scanner) {
	mu.Lock()
	defer mu.Unlock()

	for id, reg := range registrations {
		fmt.Printf("Approve account %s? (y/n): ", id)
		if !scanner.Scan() {
			return
		}
		if scanner.Text() == "y" {
			accounts[id] = &Account{ID: id, Password: reg.Password, Balance: 0, Approved: true}
			fmt.Println("Account approved.")
		} else {
			fmt.Println("Account rejected.")
		}
		delete(registrations, id) // Hapus dari pendaftaran setelah diproses
	}
}

// Fungsi untuk mencetak daftar akun
func printAccountList() {
	mu.Lock()
	defer mu.Unlock()

	fmt.Println("Account List:")
	for _, account := range accounts {
		fmt.Printf("ID: %s, Balance: %.2f, Approved: %v\n", account.ID, account.Balance, account.Approved)
	}
}

// Fungsi untuk mengecek saldo akun pengguna
func checkBalance() {
	mu.Lock()
	defer mu.Unlock()

	fmt.Printf("Balance for account %s: %.2f\n", currentUser.ID, currentUser.Balance)
}

func transferMoney(scanner *bufio.Scanner) {
	// Meminta ID akun penerima
	fmt.Print("Enter recipient account ID: ")
	if !scanner.Scan() {
		return
	}
	recipientID := scanner.Text()

	// Meminta jumlah uang yang ingin ditransfer
	fmt.Print("Enter amount to transfer: ")
	if !scanner.Scan() {
		return
	}
	amountStr := scanner.Text()
	amount, err := strconv.ParseFloat(amountStr, 64)
	// Validasi apakah jumlah uang yang dimasukkan valid atau tidak
	if err != nil || amount <= 0 {
		fmt.Println("Invalid amount.")
		return
	}

	mu.Lock()  // Mengunci agar thread-safe ketika melakukan transaksi
	defer mu.Unlock()

	// Mencari akun penerima berdasarkan ID yang diberikan
	recipient, exists := accounts[recipientID]
	if !exists {
		fmt.Println("Recipient account not found.") // Jika akun penerima tidak ditemukan
		return
	}

	// Mengecek apakah saldo mencukupi untuk transfer
	if currentUser.Balance < amount {
		fmt.Println("Insufficient funds.") // Jika saldo tidak mencukupi
		return
	}

	// Mengurangi saldo dari akun pengirim
	currentUser.Balance -= amount
	// Menambah saldo ke akun penerima
	recipient.Balance += amount

	// Menambahkan transaksi ke riwayat transaksi akun pengirim
	transaction := Transaction{
		ID:        len(currentUser.Transactions) + 1,
		AccountID: currentUser.ID,
		Type:      "Transfer",
		Amount:    amount,
		Date:      time.Now().Format(time.RFC3339),
		Details:   fmt.Sprintf("Transferred to %s", recipientID),
	}
	currentUser.Transactions = append(currentUser.Transactions, transaction)

	// Menambahkan transaksi ke riwayat transaksi akun penerima
	recipient.Transactions = append(recipient.Transactions, Transaction{
		ID:        len(recipient.Transactions) + 1,
		AccountID: recipient.ID,
		Type:      "Transfer",
		Amount:    amount,
		Date:      time.Now().Format(time.RFC3339),
		Details:   fmt.Sprintf("Received from %s", currentUser.ID),
	})

	// Konfirmasi bahwa transfer berhasil dilakukan
	fmt.Println("Transfer successful.")
}


func makePayment(scanner *bufio.Scanner) {
	fmt.Print("Enter payment type (e.g., food, phone, electricity, BPJS): ")
	if !scanner.Scan() {
		return
	}
	paymentType := scanner.Text()

	fmt.Print("Enter amount to pay: ")
	if !scanner.Scan() {
		return
	}
	amountStr := scanner.Text()
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		fmt.Println("Invalid amount.")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if currentUser.Balance < amount {
		fmt.Println("Insufficient funds.")
		return
	}

	currentUser.Balance -= amount

	transaction := Transaction{
		ID:        len(currentUser.Transactions) + 1,
		AccountID: currentUser.ID,
		Type:      "Payment",
		Amount:    amount,
		Date:      time.Now().Format(time.RFC3339),
		Details:   paymentType,
	}
	currentUser.Transactions = append(currentUser.Transactions, transaction)

	fmt.Println("Payment successful.")
}

func printTransactionHistory() {
	mu.Lock()
	defer mu.Unlock()

	fmt.Printf("Transaction history for account %s:\n", currentUser.ID)
	for _, transaction := range currentUser.Transactions {
		fmt.Printf("ID: %d, Type: %s, Amount: %.2f, Date: %s, Details: %s\n",
			transaction.ID, transaction.Type, transaction.Amount, transaction.Date, transaction.Details)
	}
}

func saveAccounts() {
	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		log.Println("Failed to marshal accounts:", err)
		return
	}

	err = os.WriteFile(accountsFilePath, data, 0644)
	if err != nil {
		log.Println("Failed to save accounts:", err)
	}
}

func loadAccounts() {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(accountsFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println("Failed to read accounts file:", err)
		}
		return
	}

	err = json.Unmarshal(data, &accounts)
	if err != nil {
		log.Println("Failed to unmarshal accounts:", err)
	}
}

func topUpBalance(scanner *bufio.Scanner) {
	fmt.Print("Enter amount to top up: ")
	if !scanner.Scan() {
		return
	}
	amountStr := scanner.Text()
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		fmt.Println("Invalid amount.")
		return
	}

	mu.Lock()
	defer mu.Unlock()

	topUpRequests = append(topUpRequests, TopUpRequest{
		ID:        nextTopUpID,
		AccountID: currentUser.ID,
		Amount:    amount,
		Date:      time.Now().Format(time.RFC3339),
		Approved:  false,
	})
	nextTopUpID++

	fmt.Println("Top up request submitted.")
}

func handleTopUpRequests(scanner *bufio.Scanner) {
	mu.Lock()
	defer mu.Unlock()

	for i, request := range topUpRequests {
		if !request.Approved {
			fmt.Printf("Approve top up request %d for account %s of amount %.2f? (y/n): ", request.ID, request.AccountID, request.Amount)
			if !scanner.Scan() {
				return
			}
			if scanner.Text() == "y" {
				account := accounts[request.AccountID]
				account.Balance += request.Amount
				topUpRequests[i].Approved = true
				fmt.Println("Top up approved.")
			} else {
				fmt.Println("Top up rejected.")
			}
		}
	}
}
