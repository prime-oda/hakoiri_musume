package board

import (
	"encoding/binary"
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

// ZobristTableBytes returns the global Zobrist table as little-endian uint64 bytes.
// Layout is [cell_index][piece_type_value], piece_type_value in 0..NumPieceTypes (0 unused),
// so the buffer has BoardSize*(NumPieceTypes+1)*8 = 1440 bytes on the standard 6x5 board.
// The JS port reads the same buffer to hash boards identically.
func ZobristTableBytes() []byte {
	buf := make([]byte, BoardSize*(NumPieceTypes+1)*8)
	off := 0
	for i := 0; i < BoardSize; i++ {
		for j := 0; j <= NumPieceTypes; j++ {
			binary.LittleEndian.PutUint64(buf[off:], globalZobristTable[i][j])
			off += 8
		}
	}
	return buf
}

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

// MirrorIndex returns the cell index for the horizontally mirrored position.
func MirrorIndex(idx int) int {
	row := idx / BoardWidth
	col := idx % BoardWidth
	return row*BoardWidth + (BoardWidth - 1 - col)
}

// HashWithMirror computes both the regular and the horizontally mirrored hash for a board.
// Since father/mother, hand/sister-in-law, etc. share their PieceType, the mirrored hash
// represents the same puzzle state up to left-right reflection.
func (zh *ZobristHasher) HashWithMirror(b *Board) (uint64, uint64) {
	var h, mh uint64
	for i, cell := range b.Grid {
		if cell != 0 {
			pt := zh.pieceTypeByID[cell]
			h ^= globalZobristTable[i][pt]
			mh ^= globalZobristTable[MirrorIndex(i)][pt]
		}
	}
	return h, mh
}

// IncrementalMirrorHash updates the mirrored hash after moving a piece.
// In the mirrored board, a piece at (x,y) with size (w,h) occupies (W-1-x-(w-1)..W-1-x, y..y+h-1).
func (zh *ZobristHasher) IncrementalMirrorHash(oldHash uint64, piece *Piece, fromX, fromY, toX, toY int) uint64 {
	pt := byte(piece.Type) + 1
	h := oldHash

	mFromX := BoardWidth - fromX - piece.Width
	mToX := BoardWidth - toX - piece.Width

	for row := 0; row < piece.Height; row++ {
		for col := 0; col < piece.Width; col++ {
			h ^= globalZobristTable[(fromY+row)*BoardWidth+(mFromX+col)][pt]
			h ^= globalZobristTable[(toY+row)*BoardWidth+(mToX+col)][pt]
		}
	}
	return h
}

// Canonical returns the smaller of (h, mh), used to identify mirror-equivalent states.
func Canonical(h, mh uint64) uint64 {
	if h < mh {
		return h
	}
	return mh
}

// IncrementalHash updates a hash after moving a piece from (fromX,fromY) to (toX,toY).
// This is O(piece_size) instead of O(BoardSize) for a full hash.
func (zh *ZobristHasher) IncrementalHash(oldHash uint64, piece *Piece, fromX, fromY, toX, toY int) uint64 {
	pt := byte(piece.Type) + 1
	h := oldHash

	// XOR out old position cells
	for row := 0; row < piece.Height; row++ {
		for col := 0; col < piece.Width; col++ {
			idx := (fromY+row)*BoardWidth + (fromX + col)
			h ^= globalZobristTable[idx][pt]
		}
	}

	// XOR in new position cells
	for row := 0; row < piece.Height; row++ {
		for col := 0; col < piece.Width; col++ {
			idx := (toY+row)*BoardWidth + (toX + col)
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

// IncrementalHash updates a hash after moving a piece.
func (sh *StateHasher) IncrementalHash(oldHash uint64, piece *Piece, fromX, fromY, toX, toY int) uint64 {
	return sh.zobrist.IncrementalHash(oldHash, piece, fromX, fromY, toX, toY)
}

// HashWithMove computes the hash after applying a 1-cell move (without modifying the board).
func (sh *StateHasher) HashWithMove(b *Board, piece *Piece, x, y int, dir Direction) uint64 {
	tempBoard := ApplyMove(*b, piece, x, y, dir)
	return sh.zobrist.Hash(&tempBoard)
}
