package sugarSplitCore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AutoSplitCommand represents a command sent via the HTTP server
type AutoSplitCommand string

const (
	CmdSplit AutoSplitCommand = "split"
	CmdReset AutoSplitCommand = "reset"
	CmdUndo  AutoSplitCommand = "undo"
	CmdSkip  AutoSplitCommand = "skip"
)

// SharedState holds a thread-safe snapshot of the run state for the HTTP server
type SharedState struct {
	mu           sync.RWMutex
	Started      bool
	Completed    bool
	CurrentSplit int
	CurrentTime  time.Duration
	GameName     string
	CategoryName string
	SplitNames   []string
	Splits       []time.Duration
}

// Update copies the current run state into the shared state snapshot
func (s *SharedState) Update(run *Run) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Started = run.Started
	s.Completed = run.Completed
	s.CurrentSplit = run.CurrentSplit
	s.CurrentTime = run.CurrentTime
	s.GameName = run.State.GameName
	s.CategoryName = run.State.CategoryName

	s.SplitNames = make([]string, len(run.State.Segments.Segments))
	for i, seg := range run.State.Segments.Segments {
		s.SplitNames[i] = seg.Name
	}

	s.Splits = make([]time.Duration, len(run.Splits))
	copy(s.Splits, run.Splits)
}

type stateResponse struct {
	Started      bool     `json:"started"`
	Completed    bool     `json:"completed"`
	CurrentSplit int      `json:"currentSplit"`
	CurrentTime  string   `json:"currentTime"`
	GameName     string   `json:"gameName"`
	CategoryName string   `json:"categoryName"`
	SplitNames   []string `json:"splitNames"`
	Splits       []string `json:"splits"`
}

// StartServer starts the auto splitter HTTP server on 127.0.0.1:<port>.
// send is called with the command when a valid request is received.
func StartServer(port int, shared *SharedState, send func(AutoSplitCommand)) {
	mux := http.NewServeMux()

	postCmd := func(cmd AutoSplitCommand) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			send(cmd)
			w.WriteHeader(http.StatusOK)
		}
	}

	mux.HandleFunc("/split", postCmd(CmdSplit))
	mux.HandleFunc("/reset", postCmd(CmdReset))
	mux.HandleFunc("/undo", postCmd(CmdUndo))
	mux.HandleFunc("/skip", postCmd(CmdSkip))

	mux.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		shared.mu.RLock()
		defer shared.mu.RUnlock()

		splits := make([]string, len(shared.Splits))
		for i, d := range shared.Splits {
			splits[i] = FormatDuration(d)
		}

		resp := stateResponse{
			Started:      shared.Started,
			Completed:    shared.Completed,
			CurrentSplit: shared.CurrentSplit,
			CurrentTime:  FormatDuration(shared.CurrentTime),
			GameName:     shared.GameName,
			CategoryName: shared.CategoryName,
			SplitNames:   shared.SplitNames,
			Splits:       splits,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	go func() {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		http.ListenAndServe(addr, mux)
	}()
}
