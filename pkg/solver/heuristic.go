// Package solver provides search algorithms for the Hakoiri Musume puzzle.
package solver

import (
	"github.com/oda/hakoiri_musume/pkg/board"
)

// GoalX and GoalY define the target position for the daughter piece (2x2).
// The goal is to move the daughter to position (2, 3).
const (
	GoalX = 2
	GoalY = 3
)

// ManhattanHeuristic computes the Manhattan distance from the daughter's
// current position to the goal position.
// This is an admissible heuristic (never overestimates the cost).
func ManhattanHeuristic(b *board.Board, daughterID board.CellType) int {
	// Find daughter's top-left position
	x, y := b.FindPiecePosition(daughterID)
	if x < 0 {
		return 9999 // Daughter not found
	}

	dx := x - GoalX
	if dx < 0 {
		dx = -dx
	}
	dy := y - GoalY
	if dy < 0 {
		dy = -dy
	}

	return dx + dy
}

// HeuristicFunc is a function type for heuristic calculators.
type HeuristicFunc func(b *board.Board, daughterID board.CellType) int

// WeightedManhattan returns a weighted Manhattan distance.
// Weight > 1 makes the search more greedy (faster but may not find optimal solution).
func WeightedManhattan(weight int) HeuristicFunc {
	return func(b *board.Board, daughterID board.CellType) int {
		return ManhattanHeuristic(b, daughterID) * weight
	}
}

// EnhancedHeuristic provides a more informed heuristic that considers
// blocking pieces between the daughter and the goal.
// This is still admissible but provides better guidance.
func EnhancedHeuristic(b *board.Board, daughterID board.CellType) int {
	x, y := b.FindPiecePosition(daughterID)
	if x < 0 {
		return 9999
	}

	// Base Manhattan distance
	dx := x - GoalX
	if dx < 0 {
		dx = -dx
	}
	dy := y - GoalY
	if dy < 0 {
		dy = -dy
	}
	h := dx + dy

	// If daughter is not at goal X position, no additional penalty
	if x != GoalX {
		return h
	}

	// If daughter is at correct X, count blocking pieces below
	// This is still admissible because each blocking piece requires
	// at least one move to clear
	if y < GoalY {
		blocking := 0
		for checkY := y + 2; checkY <= GoalY+1 && checkY < board.BoardHeight; checkY++ {
			// Check both columns of daughter's path
			for checkX := x; checkX < x+2 && checkX < board.BoardWidth; checkX++ {
				cell := b.Get(checkX, checkY)
				if cell != 0 && cell != daughterID {
					blocking++
					break // Count each blocking row once
				}
			}
		}
		// Don't double count - this is a rough lower bound
		if blocking > dy {
			blocking = dy
		}
		h += blocking
	}

	return h
}

// Heuristic is the default heuristic function used by solvers.
var Heuristic = ManhattanHeuristic
