// Hakoiri Musume Puzzle Solver
// High-performance solver for the "Hakoiri Musume no Daikazoku" puzzle
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/oda/hakoiri_musume/pkg/board"
	"github.com/oda/hakoiri_musume/pkg/solver"
)

func main() {
	// Parse command line flags
	algo := flag.String("algo", "bfs", "Search algorithm: bfs, idastar, astar, bidirectional, parallel")
	workers := flag.Int("workers", runtime.NumCPU(), "Number of parallel workers (for parallel mode)")
	maxStates := flag.Int64("max-states", 1_000_000_000, "Maximum states to explore")
	maxTime := flag.Duration("max-time", 30*time.Minute, "Maximum search time")
	bench := flag.Bool("bench", false, "Run benchmark mode")
	verbose := flag.Bool("verbose", false, "Show progress updates")
	flag.Parse()

	fmt.Println("=== 箱入り娘の大家族 Golang ソルバー ===")
	fmt.Printf("アルゴリズム: %s\n", *algo)
	fmt.Printf("最大状態数: %d\n", *maxStates)
	fmt.Printf("最大時間: %v\n", *maxTime)
	if *algo == "parallel" {
		fmt.Printf("ワーカー数: %d\n", *workers)
	}
	fmt.Println()

	// Setup initial board
	pieces := board.DefaultPieces()
	initialBoard := board.SetupInitialBoard(pieces)

	fmt.Println("初期盤面:")
	fmt.Print(initialBoard.String())
	fmt.Println()

	// Run benchmark if requested
	if *bench {
		runBenchmark(pieces, &initialBoard)
		return
	}

	// Run solver
	var result *solver.SearchResult
	var err error

	switch *algo {
	case "bfs":
		result, err = runBFS(pieces, &initialBoard, *maxStates, *maxTime, *verbose)
	case "idastar":
		result, err = runIDAStar(pieces, &initialBoard, *maxStates, *maxTime, *verbose)
	case "astar":
		result, err = runAStar(pieces, &initialBoard, *maxStates, *maxTime, *verbose)
	case "bidirectional":
		result, err = runBidirectional(pieces, &initialBoard, *maxStates, *maxTime, *verbose)
	case "parallel":
		result, err = runParallel(pieces, &initialBoard, *workers, *maxStates, *maxTime, *verbose)
	default:
		fmt.Fprintf(os.Stderr, "未知のアルゴリズム: %s\n", *algo)
		os.Exit(1)
	}

	// Print results
	fmt.Println()
	if err != nil {
		fmt.Printf("エラー: %v\n", err)
		fmt.Printf("探索状態数: %d\n", result.Stats.ExploredStates)
		fmt.Printf("最大深度: %d\n", result.Stats.MaxDepthReached)
		fmt.Printf("経過時間: %v\n", result.Stats.Elapsed)
		os.Exit(1)
	}

	fmt.Println("=== 解発見！ ===")
	fmt.Printf("手数: %d\n", result.Steps)
	fmt.Printf("探索状態数: %d\n", result.Stats.ExploredStates)
	fmt.Printf("生成状態数: %d\n", result.Stats.GeneratedStates)
	fmt.Printf("最大深度: %d\n", result.Stats.MaxDepthReached)
	fmt.Printf("経過時間: %v\n", result.Stats.Elapsed)
	fmt.Printf("速度: %.0f 状態/秒\n", result.Stats.StatesPerSecond)

	// Write solution to solve.txt
	if len(result.Path) > 0 {
		if err := writeSolution(pieces, initialBoard, result); err != nil {
			fmt.Fprintf(os.Stderr, "solve.txt 書き込みエラー: %v\n", err)
		} else {
			fmt.Println("解答手順を solve.txt に出力しました")
		}
	}
}

func runBFS(pieces []board.Piece, initial *board.Board, maxStates int64, maxTime time.Duration, verbose bool) (*solver.SearchResult, error) {
	s := solver.NewBFSSolver(pieces)
	s.MaxStates = maxStates
	s.MaxTime = maxTime

	if verbose {
		s.OnProgress = func(stats solver.SolverStats) {
			fmt.Printf("\r探索中... 状態: %d, 深度: %d, 時間: %v, 速度: %.0f/秒",
				stats.ExploredStates, stats.MaxDepthReached, stats.Elapsed.Round(time.Second), stats.StatesPerSecond)
		}
	}

	result, err := s.Solve(initial)
	if verbose {
		fmt.Println()
	}

	if result == nil {
		result = &solver.SearchResult{}
	}
	return result, err
}

func runIDAStar(pieces []board.Piece, initial *board.Board, maxStates int64, maxTime time.Duration, verbose bool) (*solver.SearchResult, error) {
	s := solver.NewIDAStarSolver(pieces)
	s.MaxStates = maxStates
	s.MaxTime = maxTime

	if verbose {
		s.OnProgress = func(stats solver.SolverStats, threshold int) {
			fmt.Printf("\rIDA* 閾値: %d, 状態: %d, 深度: %d, 時間: %v, 速度: %.0f/秒",
				threshold, stats.ExploredStates, stats.MaxDepthReached, stats.Elapsed.Round(time.Second), stats.StatesPerSecond)
		}
	}

	result, err := s.Solve(initial)
	if verbose {
		fmt.Println()
	}

	if result == nil {
		result = &solver.SearchResult{}
	}
	return result, err
}

func runAStar(pieces []board.Piece, initial *board.Board, maxStates int64, maxTime time.Duration, verbose bool) (*solver.SearchResult, error) {
	s := solver.NewAStarSolver(pieces)
	s.MaxStates = maxStates
	s.MaxTime = maxTime

	if verbose {
		s.OnProgress = func(stats solver.SolverStats) {
			fmt.Printf("\rA* 探索中... 状態: %d, 深度: %d, 時間: %v, 速度: %.0f/秒",
				stats.ExploredStates, stats.MaxDepthReached, stats.Elapsed.Round(time.Second), stats.StatesPerSecond)
		}
	}

	result, err := s.Solve(initial)
	if verbose {
		fmt.Println()
	}

	if result == nil {
		result = &solver.SearchResult{}
	}
	return result, err
}

func runBidirectional(pieces []board.Piece, initial *board.Board, maxStates int64, maxTime time.Duration, verbose bool) (*solver.SearchResult, error) {
	s := solver.NewBidirectionalSolver(pieces)
	s.MaxStates = maxStates
	s.MaxTime = maxTime

	if verbose {
		s.OnProgress = func(stats solver.SolverStats, fwdSize, bwdSize int) {
			fmt.Printf("\r双方向探索中... 状態: %d, 順方向: %d, 逆方向: %d, 時間: %v",
				stats.ExploredStates, fwdSize, bwdSize, stats.Elapsed.Round(time.Second))
		}
	}

	result, err := s.Solve(initial)
	if verbose {
		fmt.Println()
	}

	if result == nil {
		result = &solver.SearchResult{}
	}
	return result, err
}

func runParallel(pieces []board.Piece, initial *board.Board, workers int, maxStates int64, maxTime time.Duration, verbose bool) (*solver.SearchResult, error) {
	s := solver.NewParallelSolver(pieces, workers)
	s.MaxStates = maxStates
	s.MaxTime = maxTime

	if verbose {
		s.OnProgress = func(stats solver.SolverStats, workerID int) {
			fmt.Printf("\r並列探索中... ワーカー %d: 状態 %d\n", workerID, stats.ExploredStates)
		}
	}

	result, err := s.Solve(initial)
	if verbose {
		fmt.Println()
	}

	if result == nil {
		result = &solver.SearchResult{}
	}
	return result, err
}

func runBenchmark(pieces []board.Piece, initial *board.Board) {
	fmt.Println("=== ベンチマーク ===\n")

	// Benchmark move generation
	fmt.Println("移動生成ベンチマーク:")
	moveGen := board.NewMoveGenerator(pieces)
	defer moveGen.Close()

	iterations := 100000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		moveGen.Generate(initial)
	}
	elapsed := time.Since(start)
	rate := float64(iterations) / elapsed.Seconds()
	fmt.Printf("  %d 回の移動生成: %v\n", iterations, elapsed)
	fmt.Printf("  速度: %.0f 回/秒\n\n", rate)

	// Benchmark Zobrist hashing
	fmt.Println("Zobristハッシュ計算ベンチマーク:")
	zobrist := board.NewZobristHasher(pieces)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		zobrist.Hash(initial)
	}
	elapsed = time.Since(start)
	rate = float64(iterations) / elapsed.Seconds()
	fmt.Printf("  %d 回のZobristハッシュ計算: %v\n", iterations, elapsed)
	fmt.Printf("  速度: %.0f 回/秒\n\n", rate)

	// Benchmark incremental Zobrist hashing
	fmt.Println("Zobrist差分ハッシュ計算ベンチマーク:")
	initialZobristHash := zobrist.Hash(initial)
	moves2 := moveGen.Generate(initial)
	if len(moves2) > 0 {
		testMove := moves2[0]
		testPiece := moveGen.GetPiece(testMove.PieceID)
		start = time.Now()
		for i := 0; i < iterations; i++ {
			zobrist.IncrementalHash(initialZobristHash, testPiece, testMove.FromX, testMove.FromY, testMove.Direction)
		}
		elapsed = time.Since(start)
		rate = float64(iterations) / elapsed.Seconds()
		fmt.Printf("  %d 回の差分ハッシュ計算: %v\n", iterations, elapsed)
		fmt.Printf("  速度: %.0f 回/秒\n\n", rate)
	}

	// Short BFS search for speed estimation
	fmt.Println("短時間BFS探索 (10秒):")
	s := solver.NewBFSSolver(pieces)
	s.MaxTime = 10 * time.Second
	s.MaxStates = 5_000_000

	result, _ := s.Solve(initial)
	if result != nil {
		fmt.Printf("  探索状態数: %d\n", result.Stats.ExploredStates)
		fmt.Printf("  最大深度: %d\n", result.Stats.MaxDepthReached)
		fmt.Printf("  速度: %.0f 状態/秒\n", result.Stats.StatesPerSecond)
	}
}

func directionName(dir board.Direction) string {
	switch dir {
	case board.DirUp:
		return "上"
	case board.DirDown:
		return "下"
	case board.DirLeft:
		return "左"
	case board.DirRight:
		return "右"
	default:
		return "不明"
	}
}

// pieceName returns the Japanese name of a piece by ID.
var pieceNames = map[board.CellType]string{
	1: "娘", 2: "父", 3: "母", 4: "大番頭",
	5: "手代", 6: "女中", 7: "番頭", 8: "祖父",
	9: "祖母", 10: "番犬", 11: "兄嫁",
	12: "丁稚1", 13: "丁稚2", 14: "丁稚3",
}

func writeSolution(pieces []board.Piece, initialBoard board.Board, result *solver.SearchResult) error {
	f, err := os.Create("solve.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "=== 箱入り娘の大家族 解答 ===\n")
	fmt.Fprintf(f, "手数: %d\n", result.Steps)
	fmt.Fprintf(f, "探索状態数: %d\n", result.Stats.ExploredStates)
	fmt.Fprintf(f, "経過時間: %v\n", result.Stats.Elapsed)
	fmt.Fprintf(f, "速度: %.0f 状態/秒\n\n", result.Stats.StatesPerSecond)

	fmt.Fprintf(f, "初期盤面:\n")
	fmt.Fprint(f, initialBoard.String())
	fmt.Fprintln(f)

	pieceMap := make(map[board.CellType]*board.Piece)
	for i := range pieces {
		pieceMap[pieces[i].ID] = &pieces[i]
	}

	currentBoard := initialBoard
	fmt.Fprintf(f, "=== 解答手順 ===\n")
	for i, move := range result.Path {
		name := pieceNames[move.PieceID]
		if name == "" {
			name = fmt.Sprintf("駒%c", 'A'+move.PieceID-1)
		}
		dirName := directionName(move.Direction)
		fmt.Fprintf(f, "手順 %3d: %s を %s へ移動\n", i+1, name, dirName)

		piece := pieceMap[move.PieceID]
		currentBoard = board.ApplyMove(currentBoard, piece, move.FromX, move.FromY, move.Direction)
	}

	fmt.Fprintf(f, "\n最終盤面:\n")
	fmt.Fprint(f, currentBoard.String())

	return nil
}
