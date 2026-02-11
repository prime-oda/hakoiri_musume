package solver

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oda/hakoiri_musume/pkg/board"
)

// ParallelSolver implements parallel search using multiple goroutines.
type ParallelSolver struct {
	Pieces      []board.Piece
	DaughterID  board.CellType
	NumWorkers  int
	MaxStates   int64
	MaxTime     time.Duration
	Algorithm   string // "idastar" or "astar"
	Heuristic   HeuristicFunc

	// Progress callback
	OnProgress  func(stats SolverStats, workerID int)
}

// NewParallelSolver creates a new parallel solver.
func NewParallelSolver(pieces []board.Piece, numWorkers int) *ParallelSolver {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	var daughterID board.CellType
	for _, p := range pieces {
		if p.IsMain {
			daughterID = p.ID
			break
		}
	}

	return &ParallelSolver{
		Pieces:     pieces,
		DaughterID: daughterID,
		NumWorkers: numWorkers,
		MaxStates:  100_000_000,
		MaxTime:    60 * time.Minute,
		Algorithm:  "idastar",
		Heuristic:  ManhattanHeuristic,
	}
}

// WorkItem represents a work unit for parallel processing.
type WorkItem struct {
	Board board.Board
	Move  board.Move
	Depth int
}

// WorkerResult represents the result from a worker.
type WorkerResult struct {
	Path   []board.Move
	Stats  SolverStats
	Found  bool
	Error  error
}

// Solve runs parallel search.
func (s *ParallelSolver) Solve(initial *board.Board) (*SearchResult, error) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), s.MaxTime)
	defer cancel()

	// Generate initial moves to distribute to workers
	moveGen := board.NewMoveGenerator(s.Pieces)
	initialMoves := moveGen.Generate(initial)
	moveGen.Close()

	if len(initialMoves) == 0 {
		// Check if initial state is goal
		if initial.IsGoal(s.DaughterID) {
			return &SearchResult{
				Path:  nil,
				Steps: 0,
				Stats: SolverStats{},
				Found: true,
			}, nil
		}
		return nil, ErrNoSolution
	}

	// Prepare work items
	workItems := make([]WorkItem, len(initialMoves))
	for i, move := range initialMoves {
		piece := getPieceByID(s.Pieces, move.PieceID)
		newBoard := board.ApplyMove(*initial, piece, move.FromX, move.FromY, move.Direction)
		workItems[i] = WorkItem{
			Board: newBoard,
			Move:  move,
			Depth: 1,
		}
	}

	// Channels for work distribution and results
	workChan := make(chan WorkItem, len(workItems))
	resultChan := make(chan WorkerResult, s.NumWorkers)

	// Shared state for early termination
	var found int32
	var totalStates int64

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.NumWorkers; i++ {
		wg.Add(1)
		go s.worker(ctx, i, workChan, resultChan, &found, &totalStates, &wg)
	}

	// Submit work
	for _, item := range workItems {
		workChan <- item
	}
	close(workChan)

	// Wait for workers in a separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var bestResult *SearchResult
	var aggregatedStats SolverStats

	for result := range resultChan {
		aggregatedStats.ExploredStates += result.Stats.ExploredStates
		aggregatedStats.GeneratedStates += result.Stats.GeneratedStates
		if result.Stats.MaxDepthReached > aggregatedStats.MaxDepthReached {
			aggregatedStats.MaxDepthReached = result.Stats.MaxDepthReached
		}

		if result.Found {
			if bestResult == nil || len(result.Path) < bestResult.Steps {
				bestResult = &SearchResult{
					Path:  result.Path,
					Steps: len(result.Path),
					Found: true,
				}
			}
		}
	}

	aggregatedStats.Elapsed = time.Since(startTime)
	aggregatedStats.StatesPerSecond = float64(aggregatedStats.ExploredStates) / aggregatedStats.Elapsed.Seconds()

	if bestResult != nil {
		bestResult.Stats = aggregatedStats
		return bestResult, nil
	}

	// Check if cancelled due to timeout
	if ctx.Err() != nil {
		return nil, ErrTimeLimitExceeded
	}

	return nil, ErrNoSolution
}

// worker processes work items.
func (s *ParallelSolver) worker(
	ctx context.Context,
	workerID int,
	workChan <-chan WorkItem,
	resultChan chan<- WorkerResult,
	found *int32,
	totalStates *int64,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for item := range workChan {
		// Check if solution already found
		if atomic.LoadInt32(found) != 0 {
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		var result WorkerResult

		switch s.Algorithm {
		case "astar":
			result = s.runAStarWorker(ctx, &item, found, totalStates)
		default: // "idastar"
			result = s.runIDAStarWorker(ctx, &item, found, totalStates)
		}

		// Prepend the initial move to the path
		if result.Found && result.Path != nil {
			fullPath := make([]board.Move, len(result.Path)+1)
			fullPath[0] = item.Move
			copy(fullPath[1:], result.Path)
			result.Path = fullPath
		}

		resultChan <- result

		if result.Found {
			atomic.StoreInt32(found, 1)
		}
	}
}

// runIDAStarWorker runs IDA* search for a single work item.
func (s *ParallelSolver) runIDAStarWorker(
	ctx context.Context,
	item *WorkItem,
	found *int32,
	totalStates *int64,
) WorkerResult {
	solver := NewIDAStarSolver(s.Pieces)
	solver.MaxStates = s.MaxStates / int64(s.NumWorkers)
	solver.MaxTime = s.MaxTime
	solver.Heuristic = s.Heuristic

	// Check for early termination
	originalSearch := solver.search
	_ = originalSearch // Keep reference for potential use

	result, err := solver.Solve(&item.Board)

	if err != nil {
		return WorkerResult{
			Stats: solver.stats,
			Error: err,
		}
	}

	atomic.AddInt64(totalStates, solver.stats.ExploredStates)

	return WorkerResult{
		Path:  result.Path,
		Stats: solver.stats,
		Found: result.Found,
	}
}

// runAStarWorker runs A* search for a single work item.
func (s *ParallelSolver) runAStarWorker(
	ctx context.Context,
	item *WorkItem,
	found *int32,
	totalStates *int64,
) WorkerResult {
	solver := NewAStarSolver(s.Pieces)
	solver.MaxStates = s.MaxStates / int64(s.NumWorkers)
	solver.MaxTime = s.MaxTime
	solver.Heuristic = s.Heuristic

	result, err := solver.Solve(&item.Board)

	if err != nil {
		return WorkerResult{
			Error: err,
		}
	}

	return WorkerResult{
		Path:  result.Path,
		Stats: result.Stats,
		Found: result.Found,
	}
}

// getPieceByID returns the piece with the given ID.
func getPieceByID(pieces []board.Piece, id board.CellType) *board.Piece {
	for i := range pieces {
		if pieces[i].ID == id {
			return &pieces[i]
		}
	}
	return nil
}

// ParallelBFSSolver implements parallel BFS using lock-free techniques.
type ParallelBFSSolver struct {
	Pieces      []board.Piece
	DaughterID  board.CellType
	NumWorkers  int
	MaxStates   int64
	MaxTime     time.Duration

	OnProgress  func(stats SolverStats)
}

// NewParallelBFSSolver creates a new parallel BFS solver.
func NewParallelBFSSolver(pieces []board.Piece, numWorkers int) *ParallelBFSSolver {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	var daughterID board.CellType
	for _, p := range pieces {
		if p.IsMain {
			daughterID = p.ID
			break
		}
	}

	return &ParallelBFSSolver{
		Pieces:     pieces,
		DaughterID: daughterID,
		NumWorkers: numWorkers,
		MaxStates:  10_000_000,
		MaxTime:    30 * time.Minute,
	}
}

// Solve runs parallel BFS search.
func (s *ParallelBFSSolver) Solve(initial *board.Board) (*SearchResult, error) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), s.MaxTime)
	defer cancel()

	// Shared visited map with mutex
	visited := &sync.Map{}
	hasher := board.NewStateHasher(s.Pieces)

	initialHash := hasher.Hash(initial)
	visited.Store(initialHash, &State{Board: *initial, Path: nil, Depth: 0})

	// Check initial state
	if initial.IsGoal(s.DaughterID) {
		return &SearchResult{
			Path:  nil,
			Steps: 0,
			Stats: SolverStats{},
			Found: true,
		}, nil
	}

	// Current frontier
	var frontier []State
	frontier = append(frontier, State{Board: *initial, Path: nil, Depth: 0})

	var stats SolverStats
	var resultMu sync.Mutex
	var foundResult *SearchResult

	for len(frontier) > 0 && foundResult == nil {
		select {
		case <-ctx.Done():
			return nil, ErrTimeLimitExceeded
		default:
		}

		if stats.ExploredStates > s.MaxStates {
			return nil, ErrStateLimitExceeded
		}

		// Process frontier in parallel
		nextFrontier := make([]State, 0, len(frontier)*4)
		var nextMu sync.Mutex
		var localStats SolverStats

		// Divide frontier among workers
		chunkSize := (len(frontier) + s.NumWorkers - 1) / s.NumWorkers
		var wg sync.WaitGroup

		for i := 0; i < s.NumWorkers; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if end > len(frontier) {
				end = len(frontier)
			}
			if start >= len(frontier) {
				break
			}

			wg.Add(1)
			go func(chunk []State) {
				defer wg.Done()

				moveGen := board.NewMoveGenerator(s.Pieces)
				defer moveGen.Close()
				localHasher := board.NewStateHasher(s.Pieces)

				var workerNextFrontier []State
				var workerExplored int64

				for _, state := range chunk {
					workerExplored++

					// Check goal
					if state.Board.IsGoal(s.DaughterID) {
						resultMu.Lock()
						if foundResult == nil || len(state.Path) < foundResult.Steps {
							foundResult = &SearchResult{
								Path:  state.Path,
								Steps: len(state.Path),
								Found: true,
							}
						}
						resultMu.Unlock()
						continue
					}

					// Generate successors
					moves := moveGen.Generate(&state.Board)
					for _, move := range moves {
						piece := moveGen.GetPiece(move.PieceID)
						newBoard := board.ApplyMove(state.Board, piece, move.FromX, move.FromY, move.Direction)

						hash := localHasher.Hash(&newBoard)
						if _, loaded := visited.LoadOrStore(hash, true); !loaded {
							newPath := make([]board.Move, len(state.Path)+1)
							copy(newPath, state.Path)
							newPath[len(state.Path)] = move

							workerNextFrontier = append(workerNextFrontier, State{
								Board: newBoard,
								Path:  newPath,
								Depth: state.Depth + 1,
							})
						}
					}
				}

				// Merge results
				nextMu.Lock()
				nextFrontier = append(nextFrontier, workerNextFrontier...)
				atomic.AddInt64(&localStats.ExploredStates, workerExplored)
				nextMu.Unlock()
			}(frontier[start:end])
		}

		wg.Wait()

		stats.ExploredStates += localStats.ExploredStates
		stats.GeneratedStates += int64(len(nextFrontier))
		frontier = nextFrontier

		if s.OnProgress != nil {
			stats.Elapsed = time.Since(startTime)
			stats.StatesPerSecond = float64(stats.ExploredStates) / stats.Elapsed.Seconds()
			s.OnProgress(stats)
		}
	}

	stats.Elapsed = time.Since(startTime)
	stats.StatesPerSecond = float64(stats.ExploredStates) / stats.Elapsed.Seconds()

	if foundResult != nil {
		foundResult.Stats = stats
		return foundResult, nil
	}

	return nil, ErrNoSolution
}
