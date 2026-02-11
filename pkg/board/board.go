// Package board provides the core data structures for the Hakoiri Musume puzzle.
package board

// BoardWidth and BoardHeight define the puzzle dimensions (6x5).
const (
	BoardWidth  = 6
	BoardHeight = 5
	BoardSize   = BoardWidth * BoardHeight // 30 cells
)

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
// Goal: daughter (2x2) at position (2, 3) - bottom center.
func (b *Board) IsGoal(daughterID CellType) bool {
	// Check if daughter's top-left is at (2, 3)
	// Daughter occupies (2,3), (3,3), (2,4), (3,4)
	return b.Grid[3*BoardWidth+2] == daughterID &&
		b.Grid[3*BoardWidth+3] == daughterID &&
		b.Grid[4*BoardWidth+2] == daughterID &&
		b.Grid[4*BoardWidth+3] == daughterID
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

// DefaultPieces returns the standard piece configuration for Hakoiri Musume puzzle.
func DefaultPieces() []Piece {
	return []Piece{
		{ID: 1, Width: 2, Height: 2, Type: PieceTypeDaughter, IsMain: true},    // 娘 (2x2)
		{ID: 2, Width: 1, Height: 2, Type: PieceTypeFatherMother},              // 父 (1x2)
		{ID: 3, Width: 1, Height: 2, Type: PieceTypeFatherMother},              // 母 (1x2)
		{ID: 4, Width: 4, Height: 1, Type: PieceTypeGrandManager},              // 大番頭 (4x1)
		{ID: 5, Width: 1, Height: 1, Type: PieceTypeSmall},                     // 手代 (1x1)
		{ID: 6, Width: 2, Height: 1, Type: PieceTypeFamily},                    // 女中 (2x1)
		{ID: 7, Width: 2, Height: 1, Type: PieceTypeFamily},                    // 番頭 (2x1)
		{ID: 8, Width: 2, Height: 1, Type: PieceTypeFamily},                    // 祖父 (2x1)
		{ID: 9, Width: 2, Height: 1, Type: PieceTypeFamily},                    // 祖母 (2x1)
		{ID: 10, Width: 1, Height: 1, Type: PieceTypeSmall},                    // 番犬 (1x1)
		{ID: 11, Width: 1, Height: 1, Type: PieceTypeSmall},                    // 兄嫁 (1x1)
		{ID: 12, Width: 1, Height: 1, Type: PieceTypeSmall},                    // 丁稚1 (1x1)
		{ID: 13, Width: 1, Height: 1, Type: PieceTypeSmall},                    // 丁稚2 (1x1)
		{ID: 14, Width: 1, Height: 1, Type: PieceTypeSmall},                    // 丁稚3 (1x1)
	}
}

// SetupInitialBoard creates the standard initial board configuration.
// Layout:
//
//	空父娘娘母空
//	空父娘娘母空
//	手大大大大嫁
//	丁番頭女中丁
//	犬祖父祖母丁
func SetupInitialBoard(pieces []Piece) Board {
	var board Board

	// Create piece lookup map
	pieceMap := make(map[CellType]*Piece)
	for i := range pieces {
		pieceMap[pieces[i].ID] = &pieces[i]
	}

	// Place pieces according to the initial layout
	placements := []struct {
		id CellType
		x  int
		y  int
	}{
		{1, 2, 0},  // 娘 at (2,0)
		{2, 1, 0},  // 父 at (1,0)
		{3, 4, 0},  // 母 at (4,0)
		{4, 1, 2},  // 大番頭 at (1,2)
		{5, 0, 2},  // 手代 at (0,2)
		{6, 3, 3},  // 女中 at (3,3)
		{7, 1, 3},  // 番頭 at (1,3)
		{8, 1, 4},  // 祖父 at (1,4)
		{9, 3, 4},  // 祖母 at (3,4)
		{10, 0, 4}, // 番犬 at (0,4)
		{11, 5, 2}, // 兄嫁 at (5,2)
		{12, 0, 3}, // 丁稚1 at (0,3)
		{13, 5, 3}, // 丁稚2 at (5,3)
		{14, 5, 4}, // 丁稚3 at (5,4)
	}

	for _, p := range placements {
		if piece, ok := pieceMap[p.id]; ok {
			board.PlacePiece(piece, p.x, p.y)
		}
	}

	return board
}
