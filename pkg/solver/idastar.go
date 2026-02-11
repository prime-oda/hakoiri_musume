package solver

import (
	"time"

	"github.com/oda/hakoiri_musume/pkg/board"
)

const MaxThreshold = 9999

// IDAStarSolver implements IDA* (Iterative Deepening A*) search.
// This is memory-efficient as it doesn't store the entire search tree.
type IDAStarSolver struct {
	Pieces      []board.Piece
	DaughterID  board.CellType
	MaxDepth    int
	MaxStates   int64
	MaxTime     time.Duration
	Heuristic   HeuristicFunc

	// Internal state
	stats       SolverStats
	startTime   time.Time
	moveGen     *board.MoveGenerator
	hasher      *board.StateHasher
	visited     map[uint64]int // hash -> minimum depth at which visited
	path        []board.Move

	// Progress callback
	OnProgress  func(stats SolverStats, threshold int)
}

// NewIDAStarSolver creates a new IDA* solver.
func NewIDAStarSolver(pieces []board.Piece) *IDAStarSolver {
	var daughterID board.CellType
	for _, p := range pieces {
		if p.IsMain {
			daughterID = p.ID
			break
		}
	}

	return &IDAStarSolver{
		Pieces:     pieces,
		DaughterID: daughterID,
		MaxDepth:   1000,
		MaxStates:  100_000_000,
		MaxTime:    60 * time.Minute,
		Heuristic:  ManhattanHeuristic,
	}
}

// searchResult is used internally during the recursive search.
type searchResult struct {
	found         bool
	nextThreshold int
	path          []board.Move
}

// Solve runs IDA* search from the initial board.
func (s *IDAStarSolver) Solve(initial *board.Board) (*SearchResult, error) {
	s.startTime = time.Now()
	s.stats = SolverStats{}
	s.moveGen = board.NewMoveGenerator(s.Pieces)
	defer s.moveGen.Close()
	s.hasher = board.NewStateHasher(s.Pieces)
	s.path = make([]board.Move, 0, 256)

	// Initial threshold from heuristic
	threshold := s.Heuristic(initial, s.DaughterID)

	for threshold < MaxThreshold {
		// Reset visited map for each iteration
		// Using depth-limited transposition table
		s.visited = make(map[uint64]int)

		result := s.search(initial, 0, threshold)

		if result.found {
			s.stats.Elapsed = time.Since(s.startTime)
			s.stats.StatesPerSecond = float64(s.stats.ExploredStates) / s.stats.Elapsed.Seconds()
			return &SearchResult{
				Path:  result.path,
				Steps: len(result.path),
				Stats: s.stats,
				Found: true,
			}, nil
		}

		// No path found at this threshold, increase
		threshold = result.nextThreshold

		// Progress callback
		if s.OnProgress != nil {
			s.stats.Elapsed = time.Since(s.startTime)
			s.stats.StatesPerSecond = float64(s.stats.ExploredStates) / s.stats.Elapsed.Seconds()
			s.OnProgress(s.stats, threshold)
		}

		// Check limits
		if s.stats.ExploredStates > s.MaxStates {
			return nil, ErrStateLimitExceeded
		}
		if time.Since(s.startTime) > s.MaxTime {
			return nil, ErrTimeLimitExceeded
		}
	}

	s.stats.Elapsed = time.Since(s.startTime)
	return nil, ErrNoSolution
}

// search performs the depth-limited search.
func (s *IDAStarSolver) search(b *board.Board, g int, threshold int) searchResult {
	s.stats.ExploredStates++

	// Check limits periodically
	if s.stats.ExploredStates%100000 == 0 {
		if time.Since(s.startTime) > s.MaxTime {
			return searchResult{nextThreshold: MaxThreshold}
		}
		if s.stats.ExploredStates > s.MaxStates {
			return searchResult{nextThreshold: MaxThreshold}
		}
	}

	// Calculate f = g + h
	h := s.Heuristic(b, s.DaughterID)
	f := g + h

	// If f exceeds threshold, return
	if f > threshold {
		return searchResult{nextThreshold: f}
	}

	// Check goal
	if b.IsGoal(s.DaughterID) {
		// Copy the current path
		pathCopy := make([]board.Move, len(s.path))
		copy(pathCopy, s.path)
		return searchResult{found: true, path: pathCopy}
	}

	// Track max depth
	if g > s.stats.MaxDepthReached {
		s.stats.MaxDepthReached = g
	}

	// Transposition table check
	hash := s.hasher.Hash(b)
	if prevDepth, exists := s.visited[hash]; exists && prevDepth <= g {
		return searchResult{nextThreshold: MaxThreshold}
	}
	s.visited[hash] = g

	minNextThreshold := MaxThreshold

	// Generate moves - must copy since buffer is reused in recursive calls
	movesRef := s.moveGen.Generate(b)
	moves := make([]board.Move, len(movesRef))
	copy(moves, movesRef)

	for _, move := range moves {
		piece := s.moveGen.GetPiece(move.PieceID)

		// Apply move in place
		board.ApplyMoveInPlace(b, piece, move.FromX, move.FromY, move.Direction)
		s.path = append(s.path, move)

		// Recursive search
		result := s.search(b, g+1, threshold)

		// Undo move
		board.UndoMove(b, piece, move.FromX, move.FromY, move.Direction)
		s.path = s.path[:len(s.path)-1]

		if result.found {
			return result
		}

		if result.nextThreshold < minNextThreshold {
			minNextThreshold = result.nextThreshold
		}
	}

	return searchResult{nextThreshold: minNextThreshold}
}

// SolveWithPath is like Solve but also returns the full solution path.
func (s *IDAStarSolver) SolveWithPath(initial *board.Board) (*SearchResult, []board.Board, error) {
	result, err := s.Solve(initial)
	if err != nil {
		return result, nil, err
	}

	// Reconstruct board states from moves
	boards := make([]board.Board, len(result.Path)+1)
	boards[0] = *initial

	currentBoard := *initial
	for i, move := range result.Path {
		piece := s.moveGen.GetPiece(move.PieceID)
		currentBoard = board.ApplyMove(currentBoard, piece, move.FromX, move.FromY, move.Direction)
		boards[i+1] = currentBoard
	}

	return result, boards, nil
}
