package board

import (
	"math/rand"
)

// NumPieceTypes is the number of distinct piece types (excluding empty).
const NumPieceTypes = 5

// ZobristTable holds pre-computed random values for Zobrist hashing.
// Indexed by [cell_index][piece_type_value], where piece_type_value is PieceType+1 (0=empty, not used).
type ZobristTable [BoardSize][NumPieceTypes + 1]uint64

// initZobristTable generates a Zobrist table with deterministic random values.
func initZobristTable() ZobristTable {
	var table ZobristTable
	// Use a fixed seed for deterministic hashing across runs
	r := rand.New(rand.NewSource(0x48414B4F)) // "HAKO"
	for i := 0; i < BoardSize; i++ {
		for j := 1; j <= NumPieceTypes; j++ {
			table[i][j] = r.Uint64()
		}
	}
	return table
}

// globalZobristTable is the shared Zobrist table (initialized once).
var globalZobristTable = initZobristTable()

// ZobristHasher provides Zobrist hashing with piece-type normalization.
// By using piece TYPE (not piece ID) as the hash key, pieces of the same
// type are automatically treated as interchangeable through XOR commutativity.
type ZobristHasher struct {
	pieceTypeByID [256]byte // maps piece ID -> PieceType+1 (0 = unknown/empty)
}

// NewZobristHasher creates a new Zobrist hasher.
func NewZobristHasher(pieces []Piece) *ZobristHasher {
	zh := &ZobristHasher{}
	for i := range pieces {
		zh.pieceTypeByID[pieces[i].ID] = byte(pieces[i].Type) + 1
	}
	return zh
}

// Hash computes the full Zobrist hash for a board state.
func (zh *ZobristHasher) Hash(b *Board) uint64 {
	var h uint64
	for i, cell := range b.Grid {
		if cell != 0 {
			pt := zh.pieceTypeByID[cell]
			h ^= globalZobristTable[i][pt]
		}
	}
	return h
}

// IncrementalHash updates a hash after applying a move.
// This is O(piece_size) instead of O(BoardSize) for a full hash.
func (zh *ZobristHasher) IncrementalHash(oldHash uint64, piece *Piece, x, y int, dir Direction) uint64 {
	delta := DirectionDelta[dir]
	dx, dy := delta[0], delta[1]
	nx, ny := x+dx, y+dy
	pt := byte(piece.Type) + 1

	h := oldHash

	// XOR out old position cells
	for row := 0; row < piece.Height; row++ {
		for col := 0; col < piece.Width; col++ {
			idx := (y+row)*BoardWidth + (x + col)
			h ^= globalZobristTable[idx][pt]
		}
	}

	// XOR in new position cells
	for row := 0; row < piece.Height; row++ {
		for col := 0; col < piece.Width; col++ {
			idx := (ny+row)*BoardWidth + (nx + col)
			h ^= globalZobristTable[idx][pt]
		}
	}

	return h
}

// HashNormalizer provides normalized state hashing for puzzle states.
// It treats pieces of the same type as interchangeable to reduce the search space.
// This is a compatibility wrapper around ZobristHasher.
type HashNormalizer struct {
	zobrist *ZobristHasher
}

// NewHashNormalizer creates a new hash normalizer.
func NewHashNormalizer(pieces []Piece) *HashNormalizer {
	return &HashNormalizer{
		zobrist: NewZobristHasher(pieces),
	}
}

// Hash computes a normalized hash for the board state.
func (hn *HashNormalizer) Hash(b *Board) uint64 {
	return hn.zobrist.Hash(b)
}

// StateHasher provides efficient state hashing with reusable buffers.
type StateHasher struct {
	zobrist *ZobristHasher
}

// NewStateHasher creates a new state hasher.
func NewStateHasher(pieces []Piece) *StateHasher {
	return &StateHasher{
		zobrist: NewZobristHasher(pieces),
	}
}

// Hash computes the normalized hash for a board state.
func (sh *StateHasher) Hash(b *Board) uint64 {
	return sh.zobrist.Hash(b)
}

// IncrementalHash updates a hash after applying a move.
func (sh *StateHasher) IncrementalHash(oldHash uint64, piece *Piece, x, y int, dir Direction) uint64 {
	return sh.zobrist.IncrementalHash(oldHash, piece, x, y, dir)
}

// HashWithMove computes the hash after applying a move (without modifying the board).
func (sh *StateHasher) HashWithMove(b *Board, piece *Piece, x, y int, dir Direction) uint64 {
	tempBoard := ApplyMove(*b, piece, x, y, dir)
	return sh.zobrist.Hash(&tempBoard)
}
