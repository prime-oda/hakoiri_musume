//go:build !classic

package board

// 「箱入り娘の大家族」: 6×5 マス・15 駒の拡張版。
//
// 配置:
//
//	空父娘娘母空
//	空父娘娘母空
//	手大大大大嫁
//	丁番頭女中丁
//	犬祖父祖母丁
const (
	BoardWidth  = 6
	BoardHeight = 5
	BoardSize   = BoardWidth * BoardHeight

	// 娘 (2×2) のゴール左上座標。
	GoalDaughterX = 2
	GoalDaughterY = 3

	PuzzleName = "箱入り娘の大家族 (6×5, 15駒)"
)

// DefaultPieces returns the 15 pieces used by the extended puzzle.
func DefaultPieces() []Piece {
	return []Piece{
		{ID: 1, Width: 2, Height: 2, Type: PieceTypeDaughter, IsMain: true}, // 娘
		{ID: 2, Width: 1, Height: 2, Type: PieceTypeFatherMother},           // 父
		{ID: 3, Width: 1, Height: 2, Type: PieceTypeFatherMother},           // 母
		{ID: 4, Width: 4, Height: 1, Type: PieceTypeGrandManager},           // 大番頭
		{ID: 5, Width: 1, Height: 1, Type: PieceTypeSmall},                  // 手代
		{ID: 6, Width: 2, Height: 1, Type: PieceTypeFamily},                 // 女中
		{ID: 7, Width: 2, Height: 1, Type: PieceTypeFamily},                 // 番頭
		{ID: 8, Width: 2, Height: 1, Type: PieceTypeFamily},                 // 祖父
		{ID: 9, Width: 2, Height: 1, Type: PieceTypeFamily},                 // 祖母
		{ID: 10, Width: 1, Height: 1, Type: PieceTypeSmall},                 // 番犬
		{ID: 11, Width: 1, Height: 1, Type: PieceTypeSmall},                 // 兄嫁
		{ID: 12, Width: 1, Height: 1, Type: PieceTypeSmall},                 // 丁稚1
		{ID: 13, Width: 1, Height: 1, Type: PieceTypeSmall},                 // 丁稚2
		{ID: 14, Width: 1, Height: 1, Type: PieceTypeSmall},                 // 丁稚3
	}
}

// SetupInitialBoard creates the extended-puzzle initial layout.
func SetupInitialBoard(pieces []Piece) Board {
	var b Board
	pm := make(map[CellType]*Piece)
	for i := range pieces {
		pm[pieces[i].ID] = &pieces[i]
	}
	placements := []struct {
		id CellType
		x  int
		y  int
	}{
		{1, 2, 0},  // 娘
		{2, 1, 0},  // 父
		{3, 4, 0},  // 母
		{4, 1, 2},  // 大番頭
		{5, 0, 2},  // 手代
		{6, 3, 3},  // 女中
		{7, 1, 3},  // 番頭
		{8, 1, 4},  // 祖父
		{9, 3, 4},  // 祖母
		{10, 0, 4}, // 番犬
		{11, 5, 2}, // 兄嫁
		{12, 0, 3}, // 丁稚1
		{13, 5, 3}, // 丁稚2
		{14, 5, 4}, // 丁稚3
	}
	for _, p := range placements {
		if pc, ok := pm[p.id]; ok {
			b.PlacePiece(pc, p.x, p.y)
		}
	}
	return b
}

// PieceNames maps piece IDs to their Japanese labels (for solution output).
func PieceNames() map[CellType]string {
	return map[CellType]string{
		1: "娘", 2: "父", 3: "母", 4: "大番頭",
		5: "手代", 6: "女中", 7: "番頭", 8: "祖父",
		9: "祖母", 10: "番犬", 11: "兄嫁",
		12: "丁稚1", 13: "丁稚2", 14: "丁稚3",
	}
}
