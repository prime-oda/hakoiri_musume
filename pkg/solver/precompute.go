package solver

import (
	"time"

	"github.com/oda/hakoiri_musume/pkg/board"
)

// GoalDistanceEntry is one entry of a precomputed goal-distance table:
// the canonical (mirror-min) Zobrist hash of a state, paired with its shortest distance
// in moves to the nearest goal state (daughter at the bottom-center).
type GoalDistanceEntry struct {
	Hash     uint64
	Distance uint8
}

// frontierState is a slim BFS frontier element used by both precompute passes.
// We track only what's needed to expand to the next layer; no parent pointers.
type frontierState struct {
	Board      packedBoard
	Hash       uint64
	MirrorHash uint64
}

// BuildGoalDistanceTable runs a multi-source BFS from `seeds` (all at depth 0) and records
// the canonical hash and depth of every state reached, up to `maxDepth` inclusive.
//
// The result is a flat slice of (canonical_hash, distance) entries. Order matches BFS layer
// order (depth-ascending); callers that need binary-searchable output should sort by hash.
func BuildGoalDistanceTable(pieces []board.Piece, seeds []board.Board, maxDepth int, maxStates int64, maxTime time.Duration) ([]GoalDistanceEntry, *SolverStats, error) {
	startTime := time.Now()

	hasher := board.NewZobristHasher(pieces)
	moveGen := board.NewMoveGenerator(pieces)
	defer moveGen.Close()

	visited := NewHashSet(8_000_000)

	current := make([]frontierState, 0, len(seeds))
	entries := make([]GoalDistanceEntry, 0, 1_000_000)

	for i := range seeds {
		seed := seeds[i]
		h, mh := hasher.HashWithMirror(&seed)
		canonical := board.Canonical(h, mh)
		if !visited.Add(canonical) {
			continue
		}
		entries = append(entries, GoalDistanceEntry{Hash: canonical, Distance: 0})
		current = append(current, frontierState{
			Board:      packBoard(&seed),
			Hash:       h,
			MirrorHash: mh,
		})
	}

	var stats SolverStats
	depth := 0

	for len(current) > 0 && depth < maxDepth {
		depth++
		next := make([]frontierState, 0, len(current)*2)
		for i := range current {
			st := &current[i]
			stats.ExploredStates++

			if maxStates > 0 && stats.ExploredStates > maxStates {
				stats.Elapsed = time.Since(startTime)
				stats.MaxDepthReached = depth
				return entries, &stats, ErrStateLimitExceeded
			}
			if stats.ExploredStates%200_000 == 0 {
				stats.Elapsed = time.Since(startTime)
				if maxTime > 0 && stats.Elapsed > maxTime {
					stats.MaxDepthReached = depth
					return entries, &stats, ErrTimeLimitExceeded
				}
			}

			currentBoard := unpackBoard(&st.Board)
			moves := moveGen.Generate(&currentBoard)
			for _, move := range moves {
				piece := moveGen.GetPiece(move.PieceID)
				newHash := hasher.IncrementalHash(st.Hash, piece, move.FromX, move.FromY, move.ToX, move.ToY)
				newMirror := hasher.IncrementalMirrorHash(st.MirrorHash, piece, move.FromX, move.FromY, move.ToX, move.ToY)
				canonical := board.Canonical(newHash, newMirror)

				if visited.Add(canonical) {
					stats.GeneratedStates++
					newBoard := board.ApplyMoveTo(currentBoard, piece, move.FromX, move.FromY, move.ToX, move.ToY)
					entries = append(entries, GoalDistanceEntry{Hash: canonical, Distance: uint8(depth)})
					next = append(next, frontierState{
						Board:      packBoard(&newBoard),
						Hash:       newHash,
						MirrorHash: newMirror,
					})
				}
			}
		}
		current = next
	}

	stats.MaxDepthReached = depth
	stats.Elapsed = time.Since(startTime)
	if stats.Elapsed > 0 {
		stats.StatesPerSecond = float64(stats.ExploredStates) / stats.Elapsed.Seconds()
	}
	return entries, &stats, nil
}
