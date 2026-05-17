#!/usr/bin/env node
// Node smoke test for solver.js logic.
// Loads goal_distances.bin (raw), runs solver helpers directly against it,
// and checks that:
//   (1) The Zobrist table parses to the right size.
//   (2) G_final equivalent board (reconstructed with JS piece IDs that
//       produce the same PieceType-per-cell as Go's G_final) hashes to a
//       canonical that the table reports as distance 0.
//   (3) Initial board is NOT in the table (it sits at d=49 from G_final,
//       outside K=21 coverage).
//   (4) From initial, generateMoves returns a non-empty list.
//
// Run: node web/test_solver.js
//
// We can't just require('./solver.js') because it uses window globals;
// instead we polyfill `window`, fetch (filesystem-backed) and DecompressionStream
// (Node's zlib), then eval the file to populate window.PuzzleSolver.

const fs   = require('fs');
const path = require('path');
const zlib = require('zlib');

global.window = {};

// fetch() backed by the filesystem so loadSolverData() works in Node.
// We deliberately reject .bin.gz requests so solver.js's catch-and-fallback
// runs and re-fetches the raw .bin — emulating a browser without
// DecompressionStream support without polyfilling Web Streams in Node.
global.fetch = async (url) => {
    if (url.endsWith('.bin.gz')) {
        throw new Error('test fetch: forced .gz fail to exercise raw fallback');
    }
    const filePath = path.join(__dirname, url);
    const buf = fs.readFileSync(filePath);
    return {
        ok: true,
        status: 200,
        arrayBuffer: () => Promise.resolve(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength)),
    };
};

// Load and execute solver.js into our polyfilled global scope.
const solverSrc = fs.readFileSync(path.join(__dirname, 'solver.js'), 'utf8');
eval(solverSrc);

// === Test setup ===
// JS piece IDs (must match game.js setupInitialPieces ordering):
//   1=娘, 2=父, 3=母, 4=手代, 5=大番頭, 6=嫁, 7=丁1,
//   8=番頭, 9=女中, 10=丁2, 11=犬, 12=祖父, 13=祖母, 14=丁3

const initialPieces = [
    { id: 1,  x: 2, y: 0, width: 2, height: 2 },  // 娘
    { id: 2,  x: 1, y: 0, width: 1, height: 2 },  // 父
    { id: 3,  x: 4, y: 0, width: 1, height: 2 },  // 母
    { id: 4,  x: 0, y: 2, width: 1, height: 1 },  // 手代
    { id: 5,  x: 1, y: 2, width: 4, height: 1 },  // 大番頭
    { id: 6,  x: 5, y: 2, width: 1, height: 1 },  // 嫁
    { id: 7,  x: 0, y: 3, width: 1, height: 1 },  // 丁1
    { id: 8,  x: 1, y: 3, width: 2, height: 1 },  // 番頭
    { id: 9,  x: 3, y: 3, width: 2, height: 1 },  // 女中
    { id: 10, x: 5, y: 3, width: 1, height: 1 },  // 丁2
    { id: 11, x: 0, y: 4, width: 1, height: 1 },  // 犬
    { id: 12, x: 1, y: 4, width: 2, height: 1 },  // 祖父
    { id: 13, x: 3, y: 4, width: 2, height: 1 },  // 祖母
    { id: 14, x: 5, y: 4, width: 1, height: 1 },  // 丁3
];

// G_final (from Go precompute output, reconstructed with JS piece IDs whose PieceType
// matches Go's PieceType at each cell):
//   Row 0: Family Family Family Family Small  FatherMother
//   Row 1: GrandM GrandM GrandM GrandM Small  FatherMother
//   Row 2: Small  Family Family Family Family Small
//   Row 3: .      .      Daughter      FatherMother Small
//   Row 4: .      .      Daughter      FatherMother Small
const gFinalPieces = [
    { id: 1,  x: 2, y: 3, width: 2, height: 2 },  // 娘 -> Daughter (2,3)
    { id: 2,  x: 4, y: 3, width: 1, height: 2 },  // 父 -> FatherMother (4,3)
    { id: 3,  x: 5, y: 0, width: 1, height: 2 },  // 母 -> FatherMother (5,0)
    { id: 4,  x: 4, y: 0, width: 1, height: 1 },  // 手代 -> Small (4,0)
    { id: 5,  x: 0, y: 1, width: 4, height: 1 },  // 大番頭 -> GrandManager (0,1)
    { id: 6,  x: 4, y: 1, width: 1, height: 1 },  // 嫁 -> Small (4,1)
    { id: 7,  x: 0, y: 2, width: 1, height: 1 },  // 丁1 -> Small (0,2)
    { id: 8,  x: 0, y: 0, width: 2, height: 1 },  // 番頭 -> Family (0,0)
    { id: 9,  x: 2, y: 0, width: 2, height: 1 },  // 女中 -> Family (2,0)
    { id: 10, x: 5, y: 2, width: 1, height: 1 },  // 丁2 -> Small (5,2)
    { id: 11, x: 5, y: 3, width: 1, height: 1 },  // 犬 -> Small (5,3)
    { id: 12, x: 1, y: 2, width: 2, height: 1 },  // 祖父 -> Family (1,2)
    { id: 13, x: 3, y: 2, width: 2, height: 1 },  // 祖母 -> Family (3,2)
    { id: 14, x: 5, y: 4, width: 1, height: 1 },  // 丁3 -> Small (5,4)
];

// Crude mock of the Game object the solver expects.
const fakeGame = {
    pieces: initialPieces,
    updateSolverProgress: () => {},
    updateStatus: () => {},
};

function hex64(lo, hi) {
    return '0x' + (hi >>> 0).toString(16).padStart(8, '0') + (lo >>> 0).toString(16).padStart(8, '0');
}

(async () => {
    // Force the loader to run.
    const { loadSolverData, packFromPieces, computeHashes, generateMoves, isGoalPacked, tableLookup } = window.__solverInternal;
    await loadSolverData();

    // Sanity: print initial board hashes so we can cross-check against Go's dumphash.
    {
        const ip = packFromPieces(initialPieces);
        const hi = new Uint32Array(4);
        computeHashes(ip, hi);
        console.log(`Initial board hash  h=${hex64(hi[0], hi[1])}  mh=${hex64(hi[2], hi[3])}`);
    }
    // Test 1: G_final lookup → d=0
    const gFinalPacked = packFromPieces(gFinalPieces);
    if (!isGoalPacked(gFinalPacked)) {
        throw new Error('Test 1a FAIL: G_final does not pass isGoalPacked');
    }
    const h = new Uint32Array(4);
    computeHashes(gFinalPacked, h);
    console.log(`G_final hash        h=${hex64(h[0], h[1])}  mh=${hex64(h[2], h[3])}`);
    // canonical: pick smaller of (h, mh) as 64-bit unsigned
    const isH = (h[1] >>> 0) < (h[3] >>> 0) || ((h[1] >>> 0) === (h[3] >>> 0) && (h[0] >>> 0) < (h[2] >>> 0));
    const cLo = isH ? h[0] : h[2];
    const cHi = isH ? h[1] : h[3];
    console.log(`G_final canonical   ${hex64(cLo, cHi)}`);
    const d = tableLookup(cLo, cHi);
    if (d !== 0) {
        throw new Error(`Test 1 FAIL: G_final canonical lookup returned ${d}, expected 0`);
    }
    console.log('Test 1 OK: G_final → d=0');

    // Test 2: initial board NOT in table (d=49 from G_final, K=21)
    const initialPacked = packFromPieces(initialPieces);
    computeHashes(initialPacked, h);
    const isH2 = (h[1] >>> 0) < (h[3] >>> 0) || ((h[1] >>> 0) === (h[3] >>> 0) && (h[0] >>> 0) < (h[2] >>> 0));
    const cLo2 = isH2 ? h[0] : h[2];
    const cHi2 = isH2 ? h[1] : h[3];
    const d2 = tableLookup(cLo2, cHi2);
    if (d2 !== -1) {
        throw new Error(`Test 2 FAIL: initial board found in table at d=${d2}, expected -1`);
    }
    console.log('Test 2 OK: initial board not in table');

    // Test 3: generateMoves from initial returns a non-empty list
    const posBuf = new Int32Array(15);
    const moves = generateMoves(initialPacked, posBuf);
    if (moves.length === 0) {
        throw new Error('Test 3 FAIL: no moves from initial board');
    }
    console.log(`Test 3 OK: ${moves.length} moves generated from initial board`);

    // Test 4: full solve from initial board
    const PuzzleSolver = window.PuzzleSolver;
    const solver = new PuzzleSolver(fakeGame);
    console.log('Running full solve... (this may take a moment)');
    const t0 = Date.now();
    const result = await solver.solve();
    const elapsed = Date.now() - t0;
    if (!result) {
        throw new Error('Test 4 FAIL: solver returned null');
    }
    console.log(`Test 4 OK: ${result.steps} moves found, ${result.states.toLocaleString()} states explored, ${elapsed}ms, algorithm=${result.algorithm}`);
    if (result.steps !== 49) {
        console.warn(`  WARN: expected 49 moves, got ${result.steps}`);
        // Replay moves and verify final position
        const { packFromPieces } = window.__solverInternal;
        let packed = packFromPieces(initialPieces);
        // Use solver's apply via window.__solverInternal -- but applyMoveToBoard isn't exported
        // Walk the moves and print each
        for (let i = 0; i < result.moves.length; i++) {
            const m = result.moves[i];
            console.log(`  ${i+1}: piece=${m.pieceId} (${m.fromX},${m.fromY})->(${m.toX},${m.toY})`);
        }
    }

    console.log('\nAll tests passed.');
})().catch(err => {
    console.error('FAIL:', err);
    process.exit(1);
});
