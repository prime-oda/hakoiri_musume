package board

import "sync"

// Direction represents a movement direction.
type Direction int

const (
	DirUp Direction = iota
	DirDown
	DirLeft
	DirRight
)

// DirectionDelta maps direction to (dx, dy) offsets.
var DirectionDelta = [4][2]int{
	DirUp:    {0, -1},
	DirDown:  {0, 1},
	DirLeft:  {-1, 0},
	DirRight: {1, 0},
}

// Move represents a piece movement.
type Move struct {
	PieceID   CellType
	FromX     int
	FromY     int
	ToX       int
	ToY       int
	Direction Direction
}

// MoveBuffer is a reusable buffer for move generation.
type MoveBuffer struct {
	Moves []Move
}

// Reset clears the buffer for reuse.
func (mb *MoveBuffer) Reset() {
	mb.Moves = mb.Moves[:0]
}

// movePool provides reusable move buffers to reduce allocations.
var movePool = sync.Pool{
	New: func() interface{} {
		return &MoveBuffer{Moves: make([]Move, 0, 64)}
	},
}

// GetMoveBuffer retrieves a move buffer from the pool.
func GetMoveBuffer() *MoveBuffer {
	return movePool.Get().(*MoveBuffer)
}

// PutMoveBuffer returns a move buffer to the pool.
func PutMoveBuffer(mb *MoveBuffer) {
	mb.Reset()
	movePool.Put(mb)
}

// CanMove checks if a piece can move in the given direction.
// This is an optimized inline check without function call overhead.
func CanMove(b *Board, piece *Piece, x, y int, dir Direction) bool {
	delta := DirectionDelta[dir]
	dx, dy := delta[0], delta[1]
	nx, ny := x+dx, y+dy

	// Bounds check
	if nx < 0 || ny < 0 || nx+piece.Width > BoardWidth || ny+piece.Height > BoardHeight {
		return false
	}

	// Collision check - only check cells that will be newly occupied
	switch dir {
	case DirUp:
		// Check top row of destination
		for i := 0; i < piece.Width; i++ {
			cell := b.Grid[ny*BoardWidth+(nx+i)]
			if cell != 0 && cell != piece.ID {
				return false
			}
		}
	case DirDown:
		// Check bottom row of destination
		bottomY := ny + piece.Height - 1
		for i := 0; i < piece.Width; i++ {
			cell := b.Grid[bottomY*BoardWidth+(nx+i)]
			if cell != 0 && cell != piece.ID {
				return false
			}
		}
	case DirLeft:
		// Check left column of destination
		for i := 0; i < piece.Height; i++ {
			cell := b.Grid[(ny+i)*BoardWidth+nx]
			if cell != 0 && cell != piece.ID {
				return false
			}
		}
	case DirRight:
		// Check right column of destination
		rightX := nx + piece.Width - 1
		for i := 0; i < piece.Height; i++ {
			cell := b.Grid[(ny+i)*BoardWidth+rightX]
			if cell != 0 && cell != piece.ID {
				return false
			}
		}
	}

	return true
}

// ApplyMove applies a move to the board, returning a new board.
func ApplyMove(b Board, piece *Piece, x, y int, dir Direction) Board {
	delta := DirectionDelta[dir]
	dx, dy := delta[0], delta[1]
	nx, ny := x+dx, y+dy

	// Clear old position
	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(y+h)*BoardWidth+(x+w)] = 0
		}
	}

	// Set new position
	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(ny+h)*BoardWidth+(nx+w)] = piece.ID
		}
	}

	return b
}

// ApplyMoveInPlace applies a move to the board in place.
func ApplyMoveInPlace(b *Board, piece *Piece, x, y int, dir Direction) {
	delta := DirectionDelta[dir]
	dx, dy := delta[0], delta[1]
	nx, ny := x+dx, y+dy

	// Clear old position
	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(y+h)*BoardWidth+(x+w)] = 0
		}
	}

	// Set new position
	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(ny+h)*BoardWidth+(nx+w)] = piece.ID
		}
	}
}

// UndoMove reverses a move on the board in place.
func UndoMove(b *Board, piece *Piece, x, y int, dir Direction) {
	delta := DirectionDelta[dir]
	dx, dy := delta[0], delta[1]
	nx, ny := x+dx, y+dy

	// Clear new position
	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(ny+h)*BoardWidth+(nx+w)] = 0
		}
	}

	// Restore old position
	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(y+h)*BoardWidth+(x+w)] = piece.ID
		}
	}
}

// GenerateMoves generates all valid moves for the current board state.
// Returns a slice of moves without allocation by reusing the provided buffer.
func GenerateMoves(b *Board, pieces []Piece, buf *MoveBuffer) {
	buf.Reset()

	// Get piece positions
	positions := b.GetPiecePositions()

	// For each piece, try all 4 directions
	for i := range pieces {
		piece := &pieces[i]
		pos, ok := positions[piece.ID]
		if !ok {
			continue
		}

		for dir := DirUp; dir <= DirRight; dir++ {
			if CanMove(b, piece, pos.X, pos.Y, dir) {
				delta := DirectionDelta[dir]
				buf.Moves = append(buf.Moves, Move{
					PieceID:   piece.ID,
					FromX:     pos.X,
					FromY:     pos.Y,
					ToX:       pos.X + delta[0],
					ToY:       pos.Y + delta[1],
					Direction: dir,
				})
			}
		}
	}
}

// GenerateMovesSimple generates all valid moves without buffer pooling.
// Simpler but less efficient - use for testing or when simplicity matters.
func GenerateMovesSimple(b *Board, pieces []Piece) []Move {
	moves := make([]Move, 0, 32)

	positions := b.GetPiecePositions()

	for i := range pieces {
		piece := &pieces[i]
		pos, ok := positions[piece.ID]
		if !ok {
			continue
		}

		for dir := DirUp; dir <= DirRight; dir++ {
			if CanMove(b, piece, pos.X, pos.Y, dir) {
				delta := DirectionDelta[dir]
				moves = append(moves, Move{
					PieceID:   piece.ID,
					FromX:     pos.X,
					FromY:     pos.Y,
					ToX:       pos.X + delta[0],
					ToY:       pos.Y + delta[1],
					Direction: dir,
				})
			}
		}
	}

	return moves
}

// MoveGenerator is a stateful move generator that caches piece positions.
type MoveGenerator struct {
	pieces     []Piece
	pieceByID  map[CellType]*Piece
	buffer     *MoveBuffer
	ownBuffer  bool
}

// NewMoveGenerator creates a new move generator.
func NewMoveGenerator(pieces []Piece) *MoveGenerator {
	mg := &MoveGenerator{
		pieces:    pieces,
		pieceByID: make(map[CellType]*Piece),
		buffer:    GetMoveBuffer(),
		ownBuffer: true,
	}
	for i := range pieces {
		mg.pieceByID[pieces[i].ID] = &pieces[i]
	}
	return mg
}

// Close returns the internal buffer to the pool.
func (mg *MoveGenerator) Close() {
	if mg.ownBuffer && mg.buffer != nil {
		PutMoveBuffer(mg.buffer)
		mg.buffer = nil
	}
}

// Generate generates all valid moves for the given board.
func (mg *MoveGenerator) Generate(b *Board) []Move {
	GenerateMoves(b, mg.pieces, mg.buffer)
	return mg.buffer.Moves
}

// GetPiece returns the piece with the given ID.
func (mg *MoveGenerator) GetPiece(id CellType) *Piece {
	return mg.pieceByID[id]
}
