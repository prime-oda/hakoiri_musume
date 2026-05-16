package solver

import (
	"errors"
	"time"

	"github.com/oda/hakoiri_musume/pkg/board"
)

// Common errors
var (
	ErrNoSolution         = errors.New("no solution found")
	ErrStateLimitExceeded = errors.New("state limit exceeded")
	ErrTimeLimitExceeded  = errors.New("time limit exceeded")
	ErrCancelled          = errors.New("search cancelled")
)

// SolverStats holds statistics about the search.
type SolverStats struct {
	ExploredStates  int64
	GeneratedStates int64
	MaxDepthReached int
	Elapsed         time.Duration
	StatesPerSecond float64
}

// SearchResult holds the result of a search.
type SearchResult struct {
	Path  []board.Move
	Steps int
	Stats SolverStats
	Found bool
}

// State represents a search state with path tracking.
// Used by parallel solver and other algorithms.
type State struct {
	Board board.Board
	Path  []board.Move
	Depth int
}

// bfsState is a memory-efficient BFS state.
// Path is reconstructed via parent index rather than stored inline; the board is held in
// packed (4-bit-per-cell) form and the move in single-byte fields. Together this brings
// per-state cost down from ~104B to ~48B, halving peak memory at 28M+ states.
type bfsState struct {
	Board      packedBoard // 16 bytes (15 used) — 4-bit cells
	Hash       uint64      // 8 bytes — Zobrist hash of this state
	MirrorHash uint64      // 8 bytes — Zobrist hash of the horizontally mirrored state
	ParentIdx  int32       // 4 bytes — index in the states slice, -1 for initial
	Move       packedMove  // 6 bytes — the move that led to this state (8B with alignment)
}

// BFSSolver implements breadth-first search with Zobrist hashing.
type BFSSolver struct {
	Pieces     []board.Piece
	DaughterID board.CellType
	MaxStates  int64
	MaxTime    time.Duration

	// Progress callback (optional)
	OnProgress func(stats SolverStats)
}

// NewBFSSolver creates a new BFS solver.
func NewBFSSolver(pieces []board.Piece) *BFSSolver {
	var daughterID board.CellType
	for _, p := range pieces {
		if p.IsMain {
			daughterID = p.ID
			break
		}
	}

	return &BFSSolver{
		Pieces:     pieces,
		DaughterID: daughterID,
		MaxStates:  100_000_000,
		MaxTime:    60 * time.Minute,
	}
}

// reconstructPath builds the move path by following parent indices.
func reconstructPath(states []bfsState, goalIdx int) []board.Move {
	// Count path length
	length := 0
	for idx := goalIdx; states[idx].ParentIdx >= 0; idx = int(states[idx].ParentIdx) {
		length++
	}

	// Build path in reverse
	path := make([]board.Move, length)
	idx := goalIdx
	for i := length - 1; i >= 0; i-- {
		path[i] = states[idx].Move.toMove()
		idx = int(states[idx].ParentIdx)
	}
	return path
}

// Solve runs BFS search from the initial board using Zobrist hashing.
func (s *BFSSolver) Solve(initial *board.Board) (*SearchResult, error) {
	startTime := time.Now()

	// Initialize Zobrist hasher
	hasher := board.NewZobristHasher(s.Pieces)

	// Initialize move generator
	moveGen := board.NewMoveGenerator(s.Pieces)
	defer moveGen.Close()

	// Compute initial hash (and its horizontal mirror)
	initialHash, initialMirror := hasher.HashWithMirror(initial)

	// BFS queue using index-based approach (no slice shifting)
	states := make([]bfsState, 0, 1_000_000)
	states = append(states, bfsState{
		Board:      packBoard(initial),
		Hash:       initialHash,
		MirrorHash: initialMirror,
		ParentIdx:  -1,
	})

	// Visited set (keyed by canonical hash = min(h, mirror_h) for symmetry dedup).
	// Custom open-addressing set avoids Go map's per-entry overhead at 28M+ states.
	visited := NewHashSet(32_000_000)
	visited.Add(board.Canonical(initialHash, initialMirror))

	var stats SolverStats
	head := 0 // index of next state to process
	progressInterval := int64(100_000)

	for head < len(states) {
		// Snapshot the popped state's primitives; unpack the board into a local working copy.
		// Indexing states[head] directly inside the inner loop is unsafe: a later append may
		// reallocate the backing array and invalidate the pointer.
		currentHash := states[head].Hash
		currentMirror := states[head].MirrorHash
		parentIdx := int32(head)
		currentBoard := unpackBoard(&states[head].Board)
		head++

		stats.ExploredStates++

		// Periodic checks
		if stats.ExploredStates%progressInterval == 0 {
			stats.Elapsed = time.Since(startTime)
			if stats.Elapsed > s.MaxTime {
				return &SearchResult{Stats: stats}, ErrTimeLimitExceeded
			}
			if s.OnProgress != nil {
				stats.StatesPerSecond = float64(stats.ExploredStates) / stats.Elapsed.Seconds()
				stats.MaxDepthReached = s.currentDepth(states, head-1)
				s.OnProgress(stats)
			}
		}

		// Check state limit
		if stats.ExploredStates > s.MaxStates {
			return &SearchResult{Stats: stats}, ErrStateLimitExceeded
		}

		// Check goal
		if currentBoard.IsGoal(s.DaughterID) {
			stats.Elapsed = time.Since(startTime)
			stats.StatesPerSecond = float64(stats.ExploredStates) / stats.Elapsed.Seconds()
			path := reconstructPath(states, head-1)
			stats.MaxDepthReached = len(path)
			return &SearchResult{
				Path:  path,
				Steps: len(path),
				Stats: stats,
				Found: true,
			}, nil
		}

		// Generate and apply moves (slide moves: 1+ cells in one direction)
		moves := moveGen.Generate(&currentBoard)
		for _, move := range moves {
			piece := moveGen.GetPiece(move.PieceID)

			// Compute new hash and mirror hash incrementally
			newHash := hasher.IncrementalHash(currentHash, piece, move.FromX, move.FromY, move.ToX, move.ToY)
			newMirror := hasher.IncrementalMirrorHash(currentMirror, piece, move.FromX, move.FromY, move.ToX, move.ToY)
			canonical := board.Canonical(newHash, newMirror)

			if visited.Add(canonical) {
				stats.GeneratedStates++

				newBoard := board.ApplyMoveTo(currentBoard, piece, move.FromX, move.FromY, move.ToX, move.ToY)

				states = append(states, bfsState{
					Board:      packBoard(&newBoard),
					Hash:       newHash,
					MirrorHash: newMirror,
					ParentIdx:  parentIdx,
					Move:       packMove(move),
				})
			}
		}
	}

	stats.Elapsed = time.Since(startTime)
	return &SearchResult{Stats: stats}, ErrNoSolution
}

// currentDepth estimates the current BFS depth by tracing parent pointers.
func (s *BFSSolver) currentDepth(states []bfsState, idx int) int {
	depth := 0
	for i := idx; states[i].ParentIdx >= 0; i = int(states[i].ParentIdx) {
		depth++
	}
	return depth
}
