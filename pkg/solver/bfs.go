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
// Instead of storing the full path in each state, we store a parent index
// and reconstruct the path when the goal is found.
type bfsState struct {
	Board     board.Board
	Hash      uint64     // Zobrist hash of this state
	ParentIdx int32      // index in the states slice, -1 for initial
	Move      board.Move // the move that led to this state
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
		path[i] = states[idx].Move
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

	// Compute initial hash
	initialHash := hasher.Hash(initial)

	// BFS queue using index-based approach (no slice shifting)
	states := make([]bfsState, 0, 1_000_000)
	states = append(states, bfsState{
		Board:     *initial,
		Hash:      initialHash,
		ParentIdx: -1,
	})

	// Visited set
	visited := make(map[uint64]struct{}, 1_000_000)
	visited[initialHash] = struct{}{}

	var stats SolverStats
	head := 0 // index of next state to process
	progressInterval := int64(100_000)

	for head < len(states) {
		current := &states[head]
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
		if current.Board.IsGoal(s.DaughterID) {
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

		// Generate and apply moves
		moves := moveGen.Generate(&current.Board)
		for _, move := range moves {
			piece := moveGen.GetPiece(move.PieceID)

			// Compute new hash incrementally (O(piece_size) instead of O(BoardSize))
			newHash := hasher.IncrementalHash(current.Hash, piece, move.FromX, move.FromY, move.Direction)

			if _, exists := visited[newHash]; !exists {
				visited[newHash] = struct{}{}
				stats.GeneratedStates++

				newBoard := board.ApplyMove(current.Board, piece, move.FromX, move.FromY, move.Direction)

				states = append(states, bfsState{
					Board:     newBoard,
					Hash:      newHash,
					ParentIdx: int32(head - 1),
					Move:      move,
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
