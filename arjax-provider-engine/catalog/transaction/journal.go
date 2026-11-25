package transaction

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Journal persists transaction records atomically with file & directory fsync
// so interrupted sessions survive crashes and power loss cleanly.
type Journal struct {
	mu       sync.Mutex
	filePath string
	records  []*TransactionDTO
}

// DefaultJournalPath resolves $XDG_STATE_HOME/arjax/transactions.json safely.
func DefaultJournalPath() string {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			stateHome = filepath.Join(home, ".local", "state")
		} else {
			stateHome = os.TempDir()
		}
	}
	return filepath.Join(stateHome, "arjax", "transactions.json")
}

// NewJournal opens or initializes a transaction journal at filePath (uses DefaultJournalPath if empty).
func NewJournal(filePath string) *Journal {
	if filePath == "" {
		filePath = DefaultJournalPath()
	}
	j := &Journal{
		filePath: filePath,
		records:  make([]*TransactionDTO, 0),
	}
	_ = j.load()
	return j
}

func (j *Journal) load() error {
	j.mu.Lock()
	defer j.mu.Unlock()

	data, err := os.ReadFile(j.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var loaded []*TransactionDTO
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	j.records = loaded
	return nil
}

// Save writes records atomically using temp-file + File.Sync + Rename + parent Dir.Sync.
func (j *Journal) Save() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.saveLocked()
}

func (j *Journal) saveLocked() error {
	if j.filePath == "" {
		return nil
	}
	dirPath := filepath.Dir(j.filePath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	toSave := j.records
	if len(toSave) > 200 {
		toSave = toSave[len(toSave)-200:]
	}
	payload, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := j.filePath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(payload); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, j.filePath); err != nil {
		return err
	}
	// Sync parent directory to make rename durable across abrupt crashes
	if dir, err := os.Open(dirPath); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

// Append adds a transaction record and persists durably.
func (j *Journal) Append(tx *TransactionDTO) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.records = append(j.records, tx)
	_ = j.saveLocked()
}

// Update persists modifications to an existing tracked transaction.
func (j *Journal) Update(tx *TransactionDTO) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for i, existing := range j.records {
		if existing.ID == tx.ID {
			j.records[i] = tx
			_ = j.saveLocked()
			return
		}
	}
	j.records = append(j.records, tx)
	_ = j.saveLocked()
}

// Snapshot returns safe copies of all recorded transactions (newest first).
func (j *Journal) Snapshot() []TransactionDTO {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]TransactionDTO, 0, len(j.records))
	for i := len(j.records) - 1; i >= 0; i-- {
		cp := *j.records[i]
		cp.Logs = append([]string(nil), j.records[i].Logs...)
		out = append(out, cp)
	}
	return out
}

// Incomplete returns pointers to transactions loaded from disk that were left non-terminal.
func (j *Journal) Incomplete() []*TransactionDTO {
	j.mu.Lock()
	defer j.mu.Unlock()
	var out []*TransactionDTO
	for _, r := range j.records {
		if !r.State.IsTerminal() {
			out = append(out, r)
		}
	}
	return out
}
