//go:build classic

package board

// 古典「箱入り娘」(別名: 華容道 横刀立馬): 4 マス × 5 マス・10 駒。
//
// 配置 (空きマスは中段):
//
//	父 娘 娘 母
//	父 娘 娘 母
//	番 横 横 丁
//	番  .  . 丁
//	兵 兵 兵 兵
//
// - 1×2 縦長 4 駒 (父・母・番頭・丁稚)
// - 2×1 横長 1 駒 (一番上 = 横)
// - 1×1 4 駒 (兵)
// - 2×2 1 駒 (娘)
// 計 10 駒、空きマス 2。娘 (2×2) を盤面下中央 (1,3) から脱出させる。
const (
	BoardWidth  = 4
	BoardHeight = 5
	BoardSize   = BoardWidth * BoardHeight

	GoalDaughterX = 1
	GoalDaughterY = 3

	PuzzleName = "箱入り娘 (4×5, 10駒・古典)"
)

func DefaultPieces() []Piece {
	return []Piece{
		{ID: 1, Width: 2, Height: 2, Type: PieceTypeDaughter, IsMain: true}, // 娘
		{ID: 2, Width: 1, Height: 2, Type: PieceTypeFatherMother},           // 父
		{ID: 3, Width: 1, Height: 2, Type: PieceTypeFatherMother},           // 母
		{ID: 4, Width: 1, Height: 2, Type: PieceTypeFatherMother},           // 番頭
		{ID: 5, Width: 1, Height: 2, Type: PieceTypeFatherMother},           // 丁稚
		{ID: 6, Width: 2, Height: 1, Type: PieceTypeFamily},                 // 一番上 (横長)
		{ID: 7, Width: 1, Height: 1, Type: PieceTypeSmall},                  // 兵1
		{ID: 8, Width: 1, Height: 1, Type: PieceTypeSmall},                  // 兵2
		{ID: 9, Width: 1, Height: 1, Type: PieceTypeSmall},                  // 兵3
		{ID: 10, Width: 1, Height: 1, Type: PieceTypeSmall},                 // 兵4
	}
}

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
		{1, 1, 0},  // 娘
		{2, 0, 0},  // 父
		{3, 3, 0},  // 母
		{4, 0, 2},  // 番頭
		{5, 3, 2},  // 丁稚
		{6, 1, 2},  // 一番上
		{7, 0, 4},  // 兵1
		{8, 1, 4},  // 兵2
		{9, 2, 4},  // 兵3
		{10, 3, 4}, // 兵4
	}
	for _, p := range placements {
		if pc, ok := pm[p.id]; ok {
			b.PlacePiece(pc, p.x, p.y)
		}
	}
	return b
}

func PieceNames() map[CellType]string {
	return map[CellType]string{
		1: "娘", 2: "父", 3: "母", 4: "番頭", 5: "丁稚",
		6: "一番上", 7: "兵1", 8: "兵2", 9: "兵3", 10: "兵4",
	}
}
