package solver

import (
	"container/heap"
	"time"

	"github.com/oda/hakoiri_musume/pkg/board"
)

// AStarNode represents a node in the A* search.
type AStarNode struct {
	Board board.Board
	Path  []board.Move
	GCost int // Cost from start
	HCost int // Heuristic cost
	FCost int // Total cost (G + H)
	Hash  uint64
	index int // Index in heap (for heap.Interface)
}

// NodeHeap implements heap.Interface for A* priority queue.
type NodeHeap []*AStarNode

func (h NodeHeap) Len() int           { return len(h) }
func (h NodeHeap) Less(i, j int) bool { return h[i].FCost < h[j].FCost }
func (h NodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *NodeHeap) Push(x interface{}) {
	n := len(*h)
	node := x.(*AStarNode)
	node.index = n
	*h = append(*h, node)
}

func (h *NodeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	node := old[n-1]
	old[n-1] = nil // Avoid memory leak
	node.index = -1
	*h = old[0 : n-1]
	return node
}

// AStarSolver implements A* search.
type AStarSolver struct {
	Pieces      []board.Piece
	DaughterID  board.CellType
	MaxStates   int64
	MaxTime     time.Duration
	Heuristic   HeuristicFunc

	// Progress callback
	OnProgress  func(stats SolverStats)
}

// NewAStarSolver creates a new A* solver.
func NewAStarSolver(pieces []board.Piece) *AStarSolver {
	var daughterID board.CellType
	for _, p := range pieces {
		if p.IsMain {
			daughterID = p.ID
			break
		}
	}

	return &AStarSolver{
		Pieces:     pieces,
		DaughterID: daughterID,
		MaxStates:  10_000_000,
		MaxTime:    30 * time.Minute,
		Heuristic:  ManhattanHeuristic,
	}
}

// Solve runs A* search from the initial board.
func (s *AStarSolver) Solve(initial *board.Board) (*SearchResult, error) {
	startTime := time.Now()
	var stats SolverStats

	// Initialize components
	hasher := board.NewStateHasher(s.Pieces)
	moveGen := board.NewMoveGenerator(s.Pieces)
	defer moveGen.Close()

	// Priority queue (open set)
	openSet := &NodeHeap{}
	heap.Init(openSet)

	// Closed set (visited states with their best G cost)
	closedSet := make(map[uint64]int)

	// Create initial node
	initialHash := hasher.Hash(initial)
	initialH := s.Heuristic(initial, s.DaughterID)
	initialNode := &AStarNode{
		Board: *initial,
		Path:  nil,
		GCost: 0,
		HCost: initialH,
		FCost: initialH,
		Hash:  initialHash,
	}
	heap.Push(openSet, initialNode)

	progressInterval := int64(10000)

	for openSet.Len() > 0 {
		// Pop best node
		current := heap.Pop(openSet).(*AStarNode)
		stats.ExploredStates++

		// Check if already visited with lower cost
		if prevG, exists := closedSet[current.Hash]; exists && prevG <= current.GCost {
			continue
		}
		closedSet[current.Hash] = current.GCost

		// Track max depth
		if current.GCost > stats.MaxDepthReached {
			stats.MaxDepthReached = current.GCost
		}

		// Check time/state limits periodically
		if stats.ExploredStates%progressInterval == 0 {
			stats.Elapsed = time.Since(startTime)
			if stats.Elapsed > s.MaxTime {
				return nil, ErrTimeLimitExceeded
			}
			if s.OnProgress != nil {
				stats.StatesPerSecond = float64(stats.ExploredStates) / stats.Elapsed.Seconds()
				s.OnProgress(stats)
			}
		}

		if stats.ExploredStates > s.MaxStates {
			return nil, ErrStateLimitExceeded
		}

		// Check goal
		if current.Board.IsGoal(s.DaughterID) {
			stats.Elapsed = time.Since(startTime)
			stats.StatesPerSecond = float64(stats.ExploredStates) / stats.Elapsed.Seconds()
			return &SearchResult{
				Path:  current.Path,
				Steps: len(current.Path),
				Stats: stats,
				Found: true,
			}, nil
		}

		// Generate successors
		moves := moveGen.Generate(&current.Board)
		for _, move := range moves {
			piece := moveGen.GetPiece(move.PieceID)
			newBoard := board.ApplyMove(current.Board, piece, move.FromX, move.FromY, move.Direction)

			newHash := hasher.Hash(&newBoard)
			newG := current.GCost + 1

			// Skip if already visited with lower cost
			if prevG, exists := closedSet[newHash]; exists && prevG <= newG {
				continue
			}

			newH := s.Heuristic(&newBoard, s.DaughterID)

			// Create new path
			newPath := make([]board.Move, len(current.Path)+1)
			copy(newPath, current.Path)
			newPath[len(current.Path)] = move

			newNode := &AStarNode{
				Board: newBoard,
				Path:  newPath,
				GCost: newG,
				HCost: newH,
				FCost: newG + newH,
				Hash:  newHash,
			}

			stats.GeneratedStates++
			heap.Push(openSet, newNode)
		}
	}

	stats.Elapsed = time.Since(startTime)
	return nil, ErrNoSolution
}
