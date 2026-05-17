// Precompute generates a goal-distance lookup table for the puzzle by:
//  1. Running the standard forward BFS from the initial board to find one goal state
//     G_final on an optimal solution path (49 moves from initial).
//  2. Running a single-source BFS from G_final out to depth K, recording the canonical
//     (mirror-min) Zobrist hash and depth of every reached state. This gives d(state, G_final),
//     an admissible upper bound on d(state, goal_set).
//
// The resulting table is consumed by the browser JS solver: at runtime it checks IsGoal
// first (covering d=0), then canonical-hashes the current state and, on a hit, shortcuts
// the forward BFS by the table's stored depth.
//
// Multi-source BFS from the full goal_set was prototyped first but produced an 8M-entry
// d=0 layer (every distinct goal-state placement of the non-daughter pieces), blowing
// past the ~10MB bundle budget. Single-source from G_final keeps the table sparse and
// the hint stays correct as long as the runtime explicitly handles IsGoal states.
package main

import (
	"compress/gzip"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/oda/hakoiri_musume/pkg/board"
	"github.com/oda/hakoiri_musume/pkg/solver"
)

const (
	fileMagic   = "HKMG"
	fileVersion = uint16(1)
	headerSize  = 16
)

func main() {
	// K=21 yields ~9MB raw / ~7.8MB gzipped on the standard initial board — under the
	// ~10MB bundle budget agreed for the browser port. Each additional depth roughly
	// adds 1.3-1.4x the previous layer's entry count.
	maxDepth := flag.Int("k", 21, "Maximum BFS depth from G_final (inclusive)")
	outRaw := flag.String("out", "web/precomputed/goal_distances.bin", "Output path for raw binary file")
	outGz := flag.String("out-gz", "web/precomputed/goal_distances.bin.gz", "Output path for gzip-compressed file (empty to skip)")
	maxTime := flag.Duration("max-time", 10*time.Minute, "Maximum runtime for each BFS phase")
	flag.Parse()

	pieces := board.DefaultPieces()
	initial := board.SetupInitialBoard(pieces)

	pieceByID := make(map[board.CellType]*board.Piece)
	for i := range pieces {
		pieceByID[pieces[i].ID] = &pieces[i]
	}

	fmt.Println("=== Phase 1: 初期→G_final の最短経路を探索 (forward BFS) ===")
	t0 := time.Now()
	bfs := solver.NewBFSSolver(pieces)
	bfs.MaxTime = *maxTime
	bfsResult, err := bfs.Solve(&initial)
	if err != nil {
		fmt.Fprintf(os.Stderr, "forward BFS エラー: %v\n", err)
		os.Exit(1)
	}
	gFinal := initial
	for _, m := range bfsResult.Path {
		gFinal = board.ApplyMoveTo(gFinal, pieceByID[m.PieceID], m.FromX, m.FromY, m.ToX, m.ToY)
	}
	fmt.Printf("最短手数: %d\n", bfsResult.Steps)
	fmt.Printf("探索状態数: %d\n", bfsResult.Stats.ExploredStates)
	fmt.Printf("G_final:\n%s\n", gFinal.String())
	fmt.Printf("経過時間: %v\n\n", time.Since(t0))

	fmt.Printf("=== Phase 2: G_final から K=%d までの単源BFS ===\n", *maxDepth)
	t1 := time.Now()
	entries, stats2, err := solver.BuildGoalDistanceTable(pieces, []board.Board{gFinal}, *maxDepth, 0, *maxTime)
	if err != nil {
		fmt.Fprintf(os.Stderr, "backward BFS エラー: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("テーブルエントリ数: %d\n", len(entries))
	fmt.Printf("探索状態数: %d\n", stats2.ExploredStates)
	fmt.Printf("最大深度: %d\n", stats2.MaxDepthReached)
	fmt.Printf("経過時間: %v\n", time.Since(t1))
	fmt.Println()

	// Layer histogram for diagnostics
	histo := make([]int, *maxDepth+1)
	for _, e := range entries {
		if int(e.Distance) < len(histo) {
			histo[e.Distance]++
		}
	}
	fmt.Println("深度別エントリ数:")
	for d, n := range histo {
		fmt.Printf("  d=%2d: %d\n", d, n)
	}
	fmt.Println()

	// Sort by hash for binary-searchable consumption
	sort.Slice(entries, func(i, j int) bool { return entries[i].Hash < entries[j].Hash })

	if err := writeRaw(*outRaw, *maxDepth, entries); err != nil {
		fmt.Fprintf(os.Stderr, "%s 書き込みエラー: %v\n", *outRaw, err)
		os.Exit(1)
	}
	fmt.Printf("書き込み: %s (%d bytes)\n", *outRaw, headerSize+9*len(entries))

	if *outGz != "" {
		gzSize, err := writeGzip(*outGz, *maxDepth, entries)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s 書き込みエラー: %v\n", *outGz, err)
			os.Exit(1)
		}
		fmt.Printf("書き込み: %s (%d bytes, 圧縮率 %.2f)\n",
			*outGz, gzSize, float64(gzSize)/float64(headerSize+9*len(entries)))
	}
}

// writeRaw serializes entries to the given path as:
//   [0..3]   magic "HKMG"
//   [4..5]   version (LE uint16)
//   [6]      maxDepth (uint8)
//   [7]      reserved (0)
//   [8..11]  entryCount (LE uint32)
//   [12..15] reserved (0)
//   [16..]   count * 8 bytes: hashes (LE uint64), sorted ascending
//   [..]     count * 1 byte:  distances (uint8), parallel to hashes
//
// SoA layout is friendly for JS BigUint64Array + Uint8Array, and lets us page in
// the smaller distance array separately if we ever need to slim load time.
func writeRaw(path string, maxDepth int, entries []solver.GoalDistanceEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeTo(f, maxDepth, entries)
}

func writeGzip(path string, maxDepth int, entries []solver.GoalDistanceEntry) (int64, error) {
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	gz, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return 0, err
	}
	if err := writeTo(gz, maxDepth, entries); err != nil {
		_ = gz.Close()
		return 0, err
	}
	if err := gz.Close(); err != nil {
		return 0, err
	}
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func writeTo(w writeFlusher, maxDepth int, entries []solver.GoalDistanceEntry) error {
	header := make([]byte, headerSize)
	copy(header[0:4], fileMagic)
	binary.LittleEndian.PutUint16(header[4:6], fileVersion)
	header[6] = byte(maxDepth)
	binary.LittleEndian.PutUint32(header[8:12], uint32(len(entries)))
	if _, err := w.Write(header); err != nil {
		return err
	}

	buf := make([]byte, 8)
	for _, e := range entries {
		binary.LittleEndian.PutUint64(buf, e.Hash)
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}
	dists := make([]byte, len(entries))
	for i, e := range entries {
		dists[i] = e.Distance
	}
	if _, err := w.Write(dists); err != nil {
		return err
	}
	return nil
}

type writeFlusher interface {
	Write(p []byte) (int, error)
}
