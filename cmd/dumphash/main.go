// Tiny utility that prints the Zobrist hash and mirror hash of the initial
// puzzle board and of the G_final state reached by the standard BFS solver.
// Used to cross-check the JS port's hashing.
package main

import (
	"fmt"

	"github.com/oda/hakoiri_musume/pkg/board"
	"github.com/oda/hakoiri_musume/pkg/solver"
)

func main() {
	pieces := board.DefaultPieces()
	pieceByID := make(map[board.CellType]*board.Piece, len(pieces))
	for i := range pieces {
		pieceByID[pieces[i].ID] = &pieces[i]
	}
	initial := board.SetupInitialBoard(pieces)
	hasher := board.NewZobristHasher(pieces)

	ih, im := hasher.HashWithMirror(&initial)
	fmt.Printf("Initial   h=0x%016x  mh=0x%016x  canonical=0x%016x\n", ih, im, board.Canonical(ih, im))

	bfs := solver.NewBFSSolver(pieces)
	res, err := bfs.Solve(&initial)
	if err != nil || !res.Found {
		fmt.Println("BFS failed:", err)
		return
	}
	cur := initial
	for _, m := range res.Path {
		cur = board.ApplyMoveTo(cur, pieceByID[m.PieceID], m.FromX, m.FromY, m.ToX, m.ToY)
	}
	gh, gm := hasher.HashWithMirror(&cur)
	fmt.Printf("G_final   h=0x%016x  mh=0x%016x  canonical=0x%016x\n", gh, gm, board.Canonical(gh, gm))
}
