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

// Distance returns how many cells this move slides.
func (m Move) Distance() int {
	dx := m.ToX - m.FromX
	if dx < 0 {
		dx = -dx
	}
	dy := m.ToY - m.FromY
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
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
		return &MoveBuffer{Moves: make([]Move, 0, 128)}
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

// CanMove checks if a piece can move in the given direction by 1 cell.
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
		for i := 0; i < piece.Width; i++ {
			cell := b.Grid[ny*BoardWidth+(nx+i)]
			if cell != 0 && cell != piece.ID {
				return false
			}
		}
	case DirDown:
		bottomY := ny + piece.Height - 1
		for i := 0; i < piece.Width; i++ {
			cell := b.Grid[bottomY*BoardWidth+(nx+i)]
			if cell != 0 && cell != piece.ID {
				return false
			}
		}
	case DirLeft:
		for i := 0; i < piece.Height; i++ {
			cell := b.Grid[(ny+i)*BoardWidth+nx]
			if cell != 0 && cell != piece.ID {
				return false
			}
		}
	case DirRight:
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

// checkLeadingEdge checks if the leading edge at distance dist is clear.
// The piece slides from (x, y) in the given direction.
// At each incremental distance, we only need to check the new frontier cells
// that the piece would newly occupy (cells outside the original piece footprint).
func checkLeadingEdge(b *Board, pieceID CellType, pieceW, pieceH, x, y int, dir Direction, dist int) bool {
	switch dir {
	case DirUp:
		row := y - dist
		for c := 0; c < pieceW; c++ {
			cell := b.Grid[row*BoardWidth+(x+c)]
			if cell != 0 && cell != pieceID {
				return false
			}
		}
	case DirDown:
		row := y + pieceH - 1 + dist
		for c := 0; c < pieceW; c++ {
			cell := b.Grid[row*BoardWidth+(x+c)]
			if cell != 0 && cell != pieceID {
				return false
			}
		}
	case DirLeft:
		col := x - dist
		for r := 0; r < pieceH; r++ {
			cell := b.Grid[(y+r)*BoardWidth+col]
			if cell != 0 && cell != pieceID {
				return false
			}
		}
	case DirRight:
		col := x + pieceW - 1 + dist
		for r := 0; r < pieceH; r++ {
			cell := b.Grid[(y+r)*BoardWidth+col]
			if cell != 0 && cell != pieceID {
				return false
			}
		}
	}
	return true
}

// maxSlideDistance returns the board dimension limit for sliding in a direction.
func maxSlideDistance(dir Direction) int {
	if dir == DirLeft || dir == DirRight {
		return BoardWidth - 1
	}
	return BoardHeight - 1
}

// GenerateMoves generates all valid slide moves for the current board state.
// A single move allows a piece to slide 1 or more cells in one direction,
// as long as all intermediate cells along the path are clear.
func GenerateMoves(b *Board, pieces []Piece, buf *MoveBuffer) {
	buf.Reset()

	positions := b.GetPiecePositions()

	for i := range pieces {
		piece := &pieces[i]
		pos, ok := positions[piece.ID]
		if !ok {
			continue
		}

		for dir := DirUp; dir <= DirRight; dir++ {
			delta := DirectionDelta[dir]
			dx, dy := delta[0], delta[1]

			for dist := 1; dist <= maxSlideDistance(dir); dist++ {
				nx, ny := pos.X+dx*dist, pos.Y+dy*dist

				// Bounds check
				if nx < 0 || ny < 0 || nx+piece.Width > BoardWidth || ny+piece.Height > BoardHeight {
					break
				}

				// Check leading edge at this distance
				if !checkLeadingEdge(b, piece.ID, piece.Width, piece.Height, pos.X, pos.Y, dir, dist) {
					break
				}

				buf.Moves = append(buf.Moves, Move{
					PieceID:   piece.ID,
					FromX:     pos.X,
					FromY:     pos.Y,
					ToX:       nx,
					ToY:       ny,
					Direction: dir,
				})
			}
		}
	}
}

// GenerateMovesSimple generates all valid slide moves without buffer pooling.
func GenerateMovesSimple(b *Board, pieces []Piece) []Move {
	moves := make([]Move, 0, 64)

	positions := b.GetPiecePositions()

	for i := range pieces {
		piece := &pieces[i]
		pos, ok := positions[piece.ID]
		if !ok {
			continue
		}

		for dir := DirUp; dir <= DirRight; dir++ {
			delta := DirectionDelta[dir]
			dx, dy := delta[0], delta[1]

			for dist := 1; dist <= maxSlideDistance(dir); dist++ {
				nx, ny := pos.X+dx*dist, pos.Y+dy*dist

				if nx < 0 || ny < 0 || nx+piece.Width > BoardWidth || ny+piece.Height > BoardHeight {
					break
				}

				if !checkLeadingEdge(b, piece.ID, piece.Width, piece.Height, pos.X, pos.Y, dir, dist) {
					break
				}

				moves = append(moves, Move{
					PieceID:   piece.ID,
					FromX:     pos.X,
					FromY:     pos.Y,
					ToX:       nx,
					ToY:       ny,
					Direction: dir,
				})
			}
		}
	}

	return moves
}

// ApplyMoveTo applies a move from (fromX, fromY) to (toX, toY), returning a new board.
func ApplyMoveTo(b Board, piece *Piece, fromX, fromY, toX, toY int) Board {
	// Clear old position
	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(fromY+h)*BoardWidth+(fromX+w)] = 0
		}
	}

	// Set new position
	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(toY+h)*BoardWidth+(toX+w)] = piece.ID
		}
	}

	return b
}

// ApplyMove applies a 1-cell move to the board, returning a new board.
// Kept for backward compatibility with other solvers.
func ApplyMove(b Board, piece *Piece, x, y int, dir Direction) Board {
	delta := DirectionDelta[dir]
	return ApplyMoveTo(b, piece, x, y, x+delta[0], y+delta[1])
}

// ApplyMoveInPlace applies a 1-cell move to the board in place.
func ApplyMoveInPlace(b *Board, piece *Piece, x, y int, dir Direction) {
	delta := DirectionDelta[dir]
	dx, dy := delta[0], delta[1]
	nx, ny := x+dx, y+dy

	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(y+h)*BoardWidth+(x+w)] = 0
		}
	}

	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(ny+h)*BoardWidth+(nx+w)] = piece.ID
		}
	}
}

// UndoMove reverses a 1-cell move on the board in place.
func UndoMove(b *Board, piece *Piece, x, y int, dir Direction) {
	delta := DirectionDelta[dir]
	dx, dy := delta[0], delta[1]
	nx, ny := x+dx, y+dy

	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(ny+h)*BoardWidth+(nx+w)] = 0
		}
	}

	for h := 0; h < piece.Height; h++ {
		for w := 0; w < piece.Width; w++ {
			b.Grid[(y+h)*BoardWidth+(x+w)] = piece.ID
		}
	}
}

// MoveGenerator is a stateful move generator that caches piece positions.
type MoveGenerator struct {
	pieces    []Piece
	pieceByID map[CellType]*Piece
	buffer    *MoveBuffer
	ownBuffer bool
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

// Generate generates all valid slide moves for the given board.
func (mg *MoveGenerator) Generate(b *Board) []Move {
	GenerateMoves(b, mg.pieces, mg.buffer)
	return mg.buffer.Moves
}

// GetPiece returns the piece with the given ID.
func (mg *MoveGenerator) GetPiece(id CellType) *Piece {
	return mg.pieceByID[id]
}
