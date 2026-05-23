// Package board provides the core data structures for the Hakoiri Musume puzzle.
//
// Board dimensions, default pieces, and the initial layout live in
// config_extended.go (default) and config_classic.go (built with -tags classic).
package board

// CellType represents the state of each cell on the board.
// 0 = empty, 1+ = piece ID
type CellType byte

// PieceType represents the shape type of a piece.
// Used for state normalization (treating same-shaped pieces as identical).
type PieceType int

const (
	PieceTypeGrandManager PieceType = iota // 大番頭 (4x1)
	PieceTypeDaughter                      // 娘 (2x2) - target piece
	PieceTypeFatherMother                  // 父・母 (1x2)
	PieceTypeFamily                        // 祖父・祖母・番頭・女中 (2x1)
	PieceTypeSmall                         // 手代・番犬・兄嫁・丁稚 (1x1)
)

// Piece represents a puzzle piece definition.
type Piece struct {
	ID     CellType  // Unique ID on the board (1+)
	Width  int       // Width in cells
	Height int       // Height in cells
	Type   PieceType // Shape type for normalization
	IsMain bool      // True if this is the target piece (daughter)
}

// Board represents the puzzle board state using a fixed-size array.
// Using a fixed-size array enables value copying and avoids heap allocation.
type Board struct {
	Grid [BoardSize]CellType
}

// Position represents a (x, y) coordinate on the board.
type Position struct {
	X, Y int
}

// Clone returns a copy of the board (value copy for fixed-size array).
func (b Board) Clone() Board {
	return b // Value copy
}

// Get returns the cell value at (x, y).
func (b *Board) Get(x, y int) CellType {
	return b.Grid[y*BoardWidth+x]
}

// Set sets the cell value at (x, y).
func (b *Board) Set(x, y int, val CellType) {
	b.Grid[y*BoardWidth+x] = val
}

// IsEmpty returns true if the cell at (x, y) is empty.
func (b *Board) IsEmpty(x, y int) bool {
	return b.Grid[y*BoardWidth+x] == 0
}

// InBounds returns true if (x, y) is within board boundaries.
func InBounds(x, y int) bool {
	return x >= 0 && x < BoardWidth && y >= 0 && y < BoardHeight
}

// FindPiecePosition returns the top-left position of a piece by its ID.
// Returns (-1, -1) if not found.
func (b *Board) FindPiecePosition(pieceID CellType) (int, int) {
	for i, cell := range b.Grid {
		if cell == pieceID {
			return i % BoardWidth, i / BoardWidth
		}
	}
	return -1, -1
}

// GetPiecePositions returns a map of piece IDs to their top-left positions.
// Used for efficient piece lookup during move generation.
func (b *Board) GetPiecePositions() map[CellType]Position {
	positions := make(map[CellType]Position)
	for i, cell := range b.Grid {
		if cell != 0 {
			if _, exists := positions[cell]; !exists {
				positions[cell] = Position{X: i % BoardWidth, Y: i / BoardWidth}
			}
		}
	}
	return positions
}

// PlacePiece places a piece on the board at the given position.
// Returns false if the placement is invalid (out of bounds or collision).
func (b *Board) PlacePiece(piece *Piece, x, y int) bool {
	// Bounds check
	if x < 0 || y < 0 || x+piece.Width > BoardWidth || y+piece.Height > BoardHeight {
		return false
	}

	// Collision check
	for dy := 0; dy < piece.Height; dy++ {
		for dx := 0; dx < piece.Width; dx++ {
			if b.Grid[(y+dy)*BoardWidth+(x+dx)] != 0 {
				return false
			}
		}
	}

	// Place piece
	for dy := 0; dy < piece.Height; dy++ {
		for dx := 0; dx < piece.Width; dx++ {
			b.Grid[(y+dy)*BoardWidth+(x+dx)] = piece.ID
		}
	}
	return true
}

// ClearPiece removes a piece from the board.
func (b *Board) ClearPiece(piece *Piece, x, y int) {
	for dy := 0; dy < piece.Height; dy++ {
		for dx := 0; dx < piece.Width; dx++ {
			idx := (y+dy)*BoardWidth + (x + dx)
			if idx >= 0 && idx < BoardSize {
				b.Grid[idx] = 0
			}
		}
	}
}

// IsGoal checks if the daughter piece is at the goal position.
// Goal cells are determined by GoalDaughterX / GoalDaughterY (see config_*.go).
func (b *Board) IsGoal(daughterID CellType) bool {
	const gx, gy = GoalDaughterX, GoalDaughterY
	return b.Grid[gy*BoardWidth+gx] == daughterID &&
		b.Grid[gy*BoardWidth+gx+1] == daughterID &&
		b.Grid[(gy+1)*BoardWidth+gx] == daughterID &&
		b.Grid[(gy+1)*BoardWidth+gx+1] == daughterID
}

// PrintBoard returns a string representation of the board for debugging.
func (b *Board) String() string {
	result := ""
	for y := 0; y < BoardHeight; y++ {
		for x := 0; x < BoardWidth; x++ {
			cell := b.Grid[y*BoardWidth+x]
			if cell == 0 {
				result += " . "
			} else {
				result += " " + string('A'+cell-1) + " "
			}
		}
		result += "\n"
	}
	return result
}

// DefaultPieces and SetupInitialBoard are defined per build tag in
// config_extended.go and config_classic.go.
