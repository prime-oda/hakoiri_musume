package solver

import "github.com/oda/hakoiri_musume/pkg/board"

// packedBoard stores the 30 cells of a Board in 15 bytes (one nibble per cell).
// We allocate 16 bytes so the value naturally aligns to 8 bytes for cache friendliness;
// only indices 0..14 are populated, the last byte is unused.
//
// Cell at index 2k is stored in the low nibble of byte k; cell at index 2k+1 in the high nibble.
type packedBoard [16]byte

func packBoard(b *board.Board) packedBoard {
	var p packedBoard
	for i := 0; i < board.BoardSize; i += 2 {
		p[i>>1] = byte(b.Grid[i]) | (byte(b.Grid[i+1]) << 4)
	}
	return p
}

func unpackBoard(p *packedBoard) board.Board {
	var b board.Board
	for i := 0; i < board.BoardSize; i += 2 {
		v := p[i>>1]
		b.Grid[i] = board.CellType(v & 0x0F)
		b.Grid[i+1] = board.CellType(v >> 4)
	}
	return b
}

// packedMove stores the same information as board.Move using single bytes, slimming
// the per-state memory cost in the BFS queue from ~48B to 6B (8B with alignment).
// Field ranges: PieceID 0..14, From/To coordinates 0..5, Direction 0..3.
type packedMove struct {
	PieceID byte
	FromX   int8
	FromY   int8
	ToX     int8
	ToY     int8
	Dir     int8
}

func packMove(m board.Move) packedMove {
	return packedMove{
		PieceID: byte(m.PieceID),
		FromX:   int8(m.FromX),
		FromY:   int8(m.FromY),
		ToX:     int8(m.ToX),
		ToY:     int8(m.ToY),
		Dir:     int8(m.Direction),
	}
}

func (pm packedMove) toMove() board.Move {
	return board.Move{
		PieceID:   board.CellType(pm.PieceID),
		FromX:     int(pm.FromX),
		FromY:     int(pm.FromY),
		ToX:       int(pm.ToX),
		ToY:       int(pm.ToY),
		Direction: board.Direction(pm.Dir),
	}
}
