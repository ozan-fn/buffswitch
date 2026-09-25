package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	boxStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	lineStyle = lipgloss.NewStyle()
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	actStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	selStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Model is the Bubble Tea state.
type Model struct {
	credsPath string

	store    *Store
	accounts []Account
	cursor   int

	status string

	confirmDelete bool   // awaiting y/n confirmation for delete
	deleteEmail   string // account pending deletion

	needLogin  bool // set when the user asked to add an account
	needLaunch bool // set when the user picked an account and wants freebuff to run

	stats *Stats

	termW int
	termH int
}

func newModel(store *Store, credsPath string, stats *Stats) *Model {
	m := &Model{store: store, credsPath: credsPath, stats: stats}
	m.refreshList()
	return m
}

// ---- Model plumbing -------------------------------------------------------

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termW, m.termH = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) View() string {
	return m.viewList()
}

// ---- keyboard -------------------------------------------------------------

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	if m.confirmDelete {
		switch key {
		case "y", "Y":
			return m.doDelete()
		case "n", "N", "esc":
			m.confirmDelete = false
			m.deleteEmail = ""
			m.status = dimStyle.Render("Hapus dibatalkan.")
		}
		return m, nil
	}

	switch key {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if len(m.accounts) > 0 {
			m.cursor = (m.cursor - 1 + len(m.accounts)) % len(m.accounts)
		}
	case "down", "j":
		if len(m.accounts) > 0 {
			m.cursor = (m.cursor + 1) % len(m.accounts)
		}
	case "enter":
		return m.doSwitch()
	case "a":
		return m, m.startAdd()
	case "r":
		m.reload()
		m.status = dimStyle.Render("Daftar dimuat ulang dari disk.")
	case "d":
		if len(m.accounts) > 0 {
			m.deleteEmail = m.accounts[m.cursor].Email
			m.confirmDelete = true
			m.status = warnStyle.Render(fmt.Sprintf("Hapus %s? (y/n)", m.deleteEmail))
		}
	}
	return m, nil
}

func (m *Model) doDelete() (tea.Model, tea.Cmd) {
	email := m.deleteEmail
	m.confirmDelete = false
	m.deleteEmail = ""
	if err := m.store.Delete(email); err != nil {
		m.status = errStyle.Render("Gagal hapus: " + err.Error())
		return m, nil
	}
	m.refreshList()
	m.status = okStyle.Render("Akun dihapus: " + email)
	return m, nil
}

func (m *Model) doSwitch() (tea.Model, tea.Cmd) {
	if len(m.accounts) == 0 {
		return m, nil
	}
	acc := m.accounts[m.cursor]
	d, ok := m.store.Default()
	if ok && d.Email == acc.Email {
		// already active; still hand off to freebuff
		m.stats.Begin(acc.Email)
		m.needLaunch = true
		return m, tea.Quit
	}
	if err := m.store.SwitchTo(acc.Email); err != nil {
		m.status = errStyle.Render("Gagal switch: " + err.Error())
		return m, nil
	}
	m.stats.Begin(acc.Email)
	m.needLaunch = true
	return m, tea.Quit
}

// startAdd parks the current default, snapshots the accounts, and quits the
// TUI so main() can hand the terminal over to `freebuff login`.
func (m *Model) startAdd() tea.Cmd {
	m.store.ParkDefault()
	if err := m.store.Save(); err != nil {
		m.status = errStyle.Render("Gagal menyimpan: " + err.Error())
		return nil
	}
	var emails []string
	for _, a := range m.store.Accounts() {
		emails = append(emails, a.Email)
	}
	token := ""
	if d, ok := m.store.Default(); ok {
		token = d.AuthToken
	}
	if err := savePreLogin(m.credsPath, emails, token); err != nil {
		m.status = errStyle.Render("Gagal menyimpan state login: " + err.Error())
		return nil
	}
	m.needLogin = true
	return tea.Quit
}

// reload re-reads the credentials file from disk.
func (m *Model) reload() {
	if st, err := Load(m.credsPath); err == nil {
		m.store = st
	}
	m.refreshList()
}

func (m *Model) refreshList() {
	prev := ""
	if m.cursor < len(m.accounts) {
		prev = m.accounts[m.cursor].Email
	}
	m.accounts = m.store.Accounts()
	// least-used first: usage count asc, then name, then email
	if m.stats != nil {
		st := m.stats
		sort.SliceStable(m.accounts, func(i, j int) bool {
			ci, cj := st.Get(m.accounts[i].Email).Count, st.Get(m.accounts[j].Email).Count
			if ci != cj {
				return ci < cj
			}
			ni, nj := strings.ToLower(m.accounts[i].Name), strings.ToLower(m.accounts[j].Name)
			if ni != nj {
				return ni < nj
			}
			return m.accounts[i].Email < m.accounts[j].Email
		})
	}
	m.cursor = 0
	for i, a := range m.accounts {
		if a.Email == prev {
			m.cursor = i
			break
		}
	}
	if len(m.accounts) == 0 {
		m.cursor = 0
	}
}

// ---- views ----------------------------------------------------------------

// innerWidth is the content width for the 80-wide window (auto-shrink).
func (m *Model) innerWidth() int {
	cw := 76
	if m.termW > 0 {
		if w := m.termW - 6; w < cw {
			cw = w
		}
		if cw < 20 {
			cw = 20
		}
	}
	return cw
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func (m *Model) render(content string) string {
	cw := m.innerWidth()
	lines := strings.Split(content, "\n")
	padded := make([]string, len(lines))
	for i, l := range lines {
		padded[i] = lineStyle.Width(cw).Render(l)
	}
	return boxStyle.Render(strings.Join(padded, "\n"))
}

func (m *Model) viewList() string {
	cw := m.innerWidth()
	var b strings.Builder

	b.WriteString(actStyle.Render("buffswitch — account selector"))
	b.WriteString("\n\n")

	if len(m.accounts) == 0 {
		b.WriteString("Belum ada akun terdaftar.\n")
		b.WriteString(dimStyle.Render("Tekan 'a' untuk login pertama kali."))
	} else {
		avail := 12
		if m.termH > 14 {
			avail = m.termH - 12
		}
		if avail < 1 {
			avail = 1
		}
		winStart := 0
		if m.cursor >= avail {
			winStart = m.cursor - avail + 1
		}
		winEnd := winStart + avail
		if winEnd > len(m.accounts) {
			winEnd = len(m.accounts)
		}

		activeEmail := ""
		if d, ok := m.store.Default(); ok {
			activeEmail = d.Email
		}
		for i := winStart; i < winEnd; i++ {
			a := m.accounts[i]
			selected := i == m.cursor
			row := fmt.Sprintf("  %s <%s>", a.Name, a.Email)
			if a.Email == activeEmail {
				row += "  (aktif)"
			}
			if u := m.stats.Get(a.Email); u.Count > 0 {
				row += "  " + u.Summary()
			}
			line := truncate(row, cw)
			if selected {
				line = "▶" + line[1:]
				b.WriteString(selStyle.Render(line))
			} else {
				b.WriteString(dimStyle.Render(line))
			}
			if i < winEnd-1 {
				b.WriteString("\n")
			}
		}
	}

	if d, ok := m.store.Default(); ok && d.Email != "" {
		b.WriteString("\n\n")
		b.WriteString(okStyle.Render(fmt.Sprintf("Aktif sekarang: %s <%s>", d.Name, d.Email)))
	}
	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(m.status)
	}
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("↑/↓ pilih · Enter aktif & jalankan freebuff · a tambah · d hapus · r reload · q keluar"))
	return m.render(b.String())
}

// ---- entry point ----------------------------------------------------------

func main() {
	credsPath := flag.String("creds", defaultCredsPath(), "path ke credentials.json")
	flag.Parse()

	stats := LoadStats(statsPath(*credsPath))

	for {
		store, err := Load(*credsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Gagal membaca %s: %v\n", *credsPath, err)
			os.Exit(1)
		}

		m := newModel(store, *credsPath, stats)
		p := tea.NewProgram(m, tea.WithAltScreen())
		final, err := p.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		mm, ok := final.(*Model)
		if !ok {
			break
		}

		if mm.needLaunch {
			// Terminal restored — hand it over to the freebuff CLI itself.
			if err := runFreebuff(); err != nil {
				fmt.Fprintln(os.Stderr, "freebuff:", err)
				os.Exit(1)
			}
			// freebuff exited — close the session started at Enter.
			if d, ok := mm.store.Default(); ok {
				stats.End(d.Email)
			}
			break
		}

		if !mm.needLogin {
			break
		}

		// Terminal is restored here — hand it over to `freebuff login`.
		pre, perr := loadPreLogin(*credsPath)
		if perr != nil {
			fmt.Fprintln(os.Stderr, "State login tidak ditemukan:", perr)
			continue
		}
		fmt.Print("\nMenjalankan freebuff login — selesaikan login di sini…\n\n")
		msg, rerr := runLoginInteractive(*credsPath, pre)
		removePreLogin(*credsPath)
		if rerr != nil {
			fmt.Fprintln(os.Stderr, "freebuff login:", rerr)
		} else {
			fmt.Println(msg)
		}
		fmt.Print("\nTekan Enter untuk kembali ke buffswitch…")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		// loop: relaunch the TUI with the refreshed account list
	}
}
