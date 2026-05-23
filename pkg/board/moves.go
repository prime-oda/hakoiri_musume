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

// Move represents a single piece movement under the continuous-move rule:
// the piece slides through empty cells to a reachable destination, changing
// direction freely along the way. Only the endpoints (From/To) matter — the
// path taken is not recorded.
type Move struct {
	PieceID CellType
	FromX   int
	FromY   int
	ToX     int
	ToY     int
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

// footprintClear reports whether a piece of size (w, h) with the given ID can
// occupy the rectangle whose top-left is (x, y). Cells holding the piece itself
// count as clear: during a continuous move the piece vacates its starting cells,
// so it may slide through them freely.
func footprintClear(b *Board, pieceID CellType, w, h, x, y int) bool {
	for row := 0; row < h; row++ {
		base := (y+row)*BoardWidth + x
		for col := 0; col < w; col++ {
			cell := b.Grid[base+col]
			if cell != 0 && cell != pieceID {
				return false
			}
		}
	}
	return true
}

// MaxPieceID is the inclusive upper bound on piece IDs the solver allocates.
// Currently piece IDs are 1..14 (see DefaultPieces); we round up to give a little headroom.
const MaxPieceID = 16

// fillPositionsByID scans the board and records the top-left (x,y) of each piece keyed by ID.
// Returns true at index i if piece i was seen on the board.
// Caller supplies stack-resident arrays so we avoid per-call allocation.
func fillPositionsByID(b *Board, positions *[MaxPieceID + 1]Position, seen *[MaxPieceID + 1]bool) {
	for i := range seen {
		seen[i] = false
	}
	for i, cell := range b.Grid {
		if cell == 0 {
			continue
		}
		id := int(cell)
		if id > MaxPieceID || seen[id] {
			continue
		}
		seen[id] = true
		positions[id] = Position{X: i % BoardWidth, Y: i / BoardWidth}
	}
}

// reachableMoves runs a flood fill over the top-left positions a piece can
// occupy, starting from (start). Each position reachable by sliding the piece
// one cell at a time through empty space — turning as often as needed — is
// emitted as one Move. The path is irrelevant; only the destination matters,
// and each destination is emitted exactly once.
func reachableMoves(b *Board, piece *Piece, start Position, emit func(Move)) {
	var visited [BoardSize]bool
	var queue [BoardSize]Position
	head, tail := 0, 0

	visited[start.Y*BoardWidth+start.X] = true
	queue[tail] = start
	tail++

	for head < tail {
		cur := queue[head]
		head++
		for dir := DirUp; dir <= DirRight; dir++ {
			delta := DirectionDelta[dir]
			nx, ny := cur.X+delta[0], cur.Y+delta[1]
			if nx < 0 || ny < 0 || nx+piece.Width > BoardWidth || ny+piece.Height > BoardHeight {
				continue
			}
			idx := ny*BoardWidth + nx
			if visited[idx] {
				continue
			}
			if !footprintClear(b, piece.ID, piece.Width, piece.Height, nx, ny) {
				continue
			}
			visited[idx] = true
			queue[tail] = Position{X: nx, Y: ny}
			tail++
			emit(Move{PieceID: piece.ID, FromX: start.X, FromY: start.Y, ToX: nx, ToY: ny})
		}
	}
}

// GenerateMoves generates all valid moves for the current board state.
// Under the continuous-move rule a single move slides one piece through empty
// cells to any reachable destination, changing direction as needed.
func GenerateMoves(b *Board, pieces []Piece, buf *MoveBuffer) {
	buf.Reset()

	var positions [MaxPieceID + 1]Position
	var seen [MaxPieceID + 1]bool
	fillPositionsByID(b, &positions, &seen)

	for i := range pieces {
		piece := &pieces[i]
		id := int(piece.ID)
		if id > MaxPieceID || !seen[id] {
			continue
		}
		reachableMoves(b, piece, positions[id], func(m Move) {
			buf.Moves = append(buf.Moves, m)
		})
	}
}

// GenerateMovesSimple generates all valid moves without buffer pooling.
func GenerateMovesSimple(b *Board, pieces []Piece) []Move {
	moves := make([]Move, 0, 64)

	positions := b.GetPiecePositions()

	for i := range pieces {
		piece := &pieces[i]
		pos, ok := positions[piece.ID]
		if !ok {
			continue
		}
		reachableMoves(b, piece, Position{X: pos.X, Y: pos.Y}, func(m Move) {
			moves = append(moves, m)
		})
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
