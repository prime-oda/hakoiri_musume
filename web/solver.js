// 箱入り娘の大家族 — 高速ソルバー (Phase 3 JS port).
//
// 設計はGo側 (pkg/board, pkg/solver) と同じ:
//   * 4-bit pack 盤面 (Uint8Array 16B、30セルを下/上ニブルで持つ)
//   * Zobrist ハッシュ + ミラー正規化 (min(h, mh)) で対称性削減
//   * カスタム open-addressing visited (Uint32Array)
//   * 並列typed-array バックエンドの BFS キュー (parent-pointer で経路復元)
//   * Goが生成した goal_distances.bin.gz (G_finalからK=21までの距離テーブル) を
//     fetchして、ヒット時は前方BFSを打ち切ってテーブル誘導で残りを歩く
//
// ハッシュは Go の Zobrist テーブルと完全一致 (テーブル自体をバイナリに同梱)。

(() => {
'use strict';

// ----- 盤面・駒定数 -----
const BOARD_W = 6;
const BOARD_H = 5;
const BOARD_SIZE = BOARD_W * BOARD_H;       // 30
const DAUGHTER_ID = 1;
const MAX_PIECE_ID = 14;
const NUM_PIECE_TYPES = 5;                  // GrandManager..Small (pkg/board と同じ)

// 駒ID -> PieceType (pkg/board.PieceType と同値):
//   0: GrandManager (4x1)
//   1: Daughter     (2x2)
//   2: FatherMother (1x2)
//   3: Family       (2x1)  -- 番頭, 女中, 祖父, 祖母
//   4: Small        (1x1)  -- 手代, 嫁, 犬, 丁1, 丁2, 丁3
// JS側のGameクラスが setupInitialPieces で 1..14 を以下の順で割り当てている:
//   1=娘 2=父 3=母 4=手代 5=大番頭 6=嫁 7=丁1 8=番頭 9=女中 10=丁2 11=犬 12=祖父 13=祖母 14=丁3
const PIECE_TYPES = new Int8Array([
    -1,  // 0: 未使用 (空セル)
    1,   // 1: 娘
    2,   // 2: 父
    2,   // 3: 母
    4,   // 4: 手代
    0,   // 5: 大番頭
    4,   // 6: 嫁
    4,   // 7: 丁1
    3,   // 8: 番頭
    3,   // 9: 女中
    4,   // 10: 丁2
    4,   // 11: 犬
    3,   // 12: 祖父
    3,   // 13: 祖母
    4,   // 14: 丁3
]);
const PIECE_W = new Int8Array([0, 2, 1, 1, 1, 4, 1, 1, 2, 2, 1, 1, 2, 2, 1]);
const PIECE_H = new Int8Array([0, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1]);

// ----- データ (Zobrist + goal-distance テーブル) ローダ -----
const DATA_URL_GZ  = 'precomputed/goal_distances.bin.gz';
const DATA_URL_RAW = 'precomputed/goal_distances.bin';

let ZOBRIST = null;     // Uint32Array(BOARD_SIZE * (NUM_PIECE_TYPES + 1) * 2)  --  [lo, hi] pairs
let GOAL_TABLE = null;  // open-addressing { hashLo, hashHi, distance, occupied, mask, cap, maxDepth, count }
let LOAD_PROMISE = null;

async function loadSolverData() {
    if (LOAD_PROMISE) return LOAD_PROMISE;
    LOAD_PROMISE = (async () => {
        let buf;
        try {
            const resp = await fetch(DATA_URL_GZ);
            if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
            if (typeof DecompressionStream !== 'undefined') {
                const stream = resp.body.pipeThrough(new DecompressionStream('gzip'));
                buf = new Uint8Array(await new Response(stream).arrayBuffer());
            } else {
                // Server may already serve with Content-Encoding: gzip
                buf = new Uint8Array(await resp.arrayBuffer());
            }
        } catch (e) {
            console.warn('gz load failed, falling back to raw:', e.message);
            const resp = await fetch(DATA_URL_RAW);
            if (!resp.ok) throw new Error(`raw fetch failed: HTTP ${resp.status}`);
            buf = new Uint8Array(await resp.arrayBuffer());
        }
        parseData(buf);
    })();
    return LOAD_PROMISE;
}

function parseData(buf) {
    const magic = String.fromCharCode(buf[0], buf[1], buf[2], buf[3]);
    if (magic !== 'HKMG') throw new Error(`Bad magic: "${magic}"`);
    const version = buf[4] | (buf[5] << 8);
    if (version !== 2) throw new Error(`Unsupported version: ${version} (expected 2)`);
    const maxDepth = buf[6];
    const dv = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
    const count = dv.getUint32(8, true);

    // Zobrist table: 30 * 6 = 180 uint64s starting at byte 16 (header is 16 bytes)
    const ZOBRIST_ENTRIES = BOARD_SIZE * (NUM_PIECE_TYPES + 1);
    const zobristOffset = 16;
    ZOBRIST = new Uint32Array(ZOBRIST_ENTRIES * 2);
    for (let i = 0; i < ZOBRIST_ENTRIES; i++) {
        ZOBRIST[i * 2]     = dv.getUint32(zobristOffset + i * 8,     true);
        ZOBRIST[i * 2 + 1] = dv.getUint32(zobristOffset + i * 8 + 4, true);
    }

    // Hashes and distances follow the Zobrist block.
    const hashesOffset    = zobristOffset + ZOBRIST_ENTRIES * 8;
    const distancesOffset = hashesOffset  + count * 8;

    // Build open-addressing distance table at ~0.5 load.
    let cap = 16;
    while (cap < count * 2) cap <<= 1;
    const mask = cap - 1;
    const hashLo  = new Uint32Array(cap);
    const hashHi  = new Uint32Array(cap);
    const distance = new Uint8Array(cap);
    const occupied = new Uint8Array(cap);
    for (let i = 0; i < count; i++) {
        const lo = dv.getUint32(hashesOffset + i * 8,     true);
        const hi = dv.getUint32(hashesOffset + i * 8 + 4, true);
        const d  = buf[distancesOffset + i];
        let idx = lo & mask;
        while (occupied[idx]) idx = (idx + 1) & mask;
        occupied[idx] = 1;
        hashLo[idx]   = lo;
        hashHi[idx]   = hi;
        distance[idx] = d;
    }
    GOAL_TABLE = { hashLo, hashHi, distance, occupied, mask, cap, maxDepth, count };
    console.log(`Goal-distance table loaded: ${count} entries, K=${maxDepth} (${(buf.length / 1048576).toFixed(2)}MB raw)`);
}

function tableLookup(lo, hi) {
    const t = GOAL_TABLE;
    if (!t) return -1;
    let idx = lo & t.mask;
    while (t.occupied[idx]) {
        if (t.hashLo[idx] === lo && t.hashHi[idx] === hi) return t.distance[idx];
        idx = (idx + 1) & t.mask;
    }
    return -1;
}

// ----- 盤面パック/アンパック -----
function getCell(packed, i)   { return (i & 1) ? (packed[i >> 1] >> 4) : (packed[i >> 1] & 0x0F); }
function setCell(packed, i, v) {
    const bi = i >> 1, b = packed[bi];
    packed[bi] = (i & 1) ? ((b & 0x0F) | ((v & 0x0F) << 4)) : ((b & 0xF0) | (v & 0x0F));
}

function packFromPieces(pieces) {
    const packed = new Uint8Array(16);
    for (const p of pieces) {
        for (let dy = 0; dy < p.height; dy++) {
            for (let dx = 0; dx < p.width; dx++) {
                setCell(packed, (p.y + dy) * BOARD_W + (p.x + dx), p.id);
            }
        }
    }
    return packed;
}

function mirrorIndex(i) {
    const row = (i / BOARD_W) | 0;
    const col = i - row * BOARD_W;
    return row * BOARD_W + (BOARD_W - 1 - col);
}

// 盤面から (hLo, hHi, mLo, mHi) を計算。Zobristテーブルの type slot は +1 オフセット
// (j=0 は未使用; pkg/board と一致)。
function computeHashes(packed, out) {
    let hLo = 0, hHi = 0, mLo = 0, mHi = 0;
    const z = ZOBRIST;
    for (let i = 0; i < BOARD_SIZE; i++) {
        const cell = getCell(packed, i);
        if (cell === 0) continue;
        const pt = PIECE_TYPES[cell] + 1;
        let off = (i * 6 + pt) * 2;
        hLo ^= z[off]; hHi ^= z[off + 1];
        const mi = mirrorIndex(i);
        off = (mi * 6 + pt) * 2;
        mLo ^= z[off]; mHi ^= z[off + 1];
    }
    out[0] = hLo; out[1] = hHi; out[2] = mLo; out[3] = mHi;
}

// 駒移動に伴う増分ハッシュ更新 (h, mh の両方)。
function applyMoveHashes(prev, out, pieceId, fromX, fromY, toX, toY) {
    const pt = PIECE_TYPES[pieceId] + 1;
    const w  = PIECE_W[pieceId];
    const h  = PIECE_H[pieceId];
    const z = ZOBRIST;
    let hLo = prev[0], hHi = prev[1], mLo = prev[2], mHi = prev[3];

    const mFromX = BOARD_W - fromX - w;
    const mToX   = BOARD_W - toX   - w;
    for (let dy = 0; dy < h; dy++) {
        const fy = (fromY + dy) * BOARD_W;
        const ty = (toY   + dy) * BOARD_W;
        for (let dx = 0; dx < w; dx++) {
            let off;
            off = ((fy + fromX + dx) * 6 + pt) * 2;
            hLo ^= z[off]; hHi ^= z[off + 1];
            off = ((ty + toX   + dx) * 6 + pt) * 2;
            hLo ^= z[off]; hHi ^= z[off + 1];
            off = ((fy + mFromX + dx) * 6 + pt) * 2;
            mLo ^= z[off]; mHi ^= z[off + 1];
            off = ((ty + mToX   + dx) * 6 + pt) * 2;
            mLo ^= z[off]; mHi ^= z[off + 1];
        }
    }
    out[0] = hLo; out[1] = hHi; out[2] = mLo; out[3] = mHi;
}

// h (=元) と mh (=ミラー) のうち、64-bit unsigned で小さい方を canonical とする。
// JS の bitwise はsigned 32bit なので、unsigned 比較は `>>> 0`。
function canonicalLo(h)  { /* h: [hLo, hHi, mLo, mHi] */
    return (h[1] >>> 0) < (h[3] >>> 0)
        || ((h[1] >>> 0) === (h[3] >>> 0) && (h[0] >>> 0) < (h[2] >>> 0))
        ? h[0] : h[2];
}
function canonicalHi(h) {
    return (h[1] >>> 0) < (h[3] >>> 0)
        || ((h[1] >>> 0) === (h[3] >>> 0) && (h[0] >>> 0) < (h[2] >>> 0))
        ? h[1] : h[3];
}

// 駒位置キャッシュ (idで引ける、未配置は -1)
function findPositions(packed, out) {
    out.fill(-1);
    for (let i = 0; i < BOARD_SIZE; i++) {
        const cell = getCell(packed, i);
        if (cell === 0 || out[cell] !== -1) continue;
        out[cell] = i;
    }
}

// 駒 (id, w, h) が左上 (x, y) を占有できるか (footprint が空きか) を検査。
// 自分自身のセルは「動けば空く」ので通過可能として扱う。
function footprintClear(packed, pieceId, w, h, x, y) {
    for (let dy = 0; dy < h; dy++) {
        const row = (y + dy) * BOARD_W;
        for (let dx = 0; dx < w; dx++) {
            const cell = getCell(packed, row + x + dx);
            if (cell !== 0 && cell !== pieceId) return false;
        }
    }
    return true;
}

const DIRS = [[0,-1],[0,1],[-1,0],[1,0]];

// flood fill 用の使い回しバッファ (単一スレッドなので再入なし)。
const _ffVisited = new Uint8Array(BOARD_SIZE);
const _ffQueue   = new Int32Array(BOARD_SIZE);

// 移動生成: {pieceId, fromX, fromY, toX, toY} の配列。
// 連続移動ルール: 各駒を空きマス内で1マスずつ滑らせ、何度曲がってもよい。
// 到達できる左上位置を flood fill で列挙し、各到達先を1手として出す。
// posBuf は呼び出し側で確保された Int32Array(15) を渡してアロケーション削減。
function generateMoves(packed, posBuf) {
    findPositions(packed, posBuf);
    const moves = [];
    for (let id = 1; id <= MAX_PIECE_ID; id++) {
        const flat = posBuf[id];
        if (flat < 0) continue;
        const px = flat % BOARD_W;
        const py = (flat / BOARD_W) | 0;
        const w  = PIECE_W[id];
        const h  = PIECE_H[id];

        _ffVisited.fill(0);
        let head = 0, tail = 0;
        _ffVisited[flat] = 1;
        _ffQueue[tail++] = flat;
        while (head < tail) {
            const cf = _ffQueue[head++];
            const cx = cf % BOARD_W;
            const cy = (cf / BOARD_W) | 0;
            for (let d = 0; d < 4; d++) {
                const nx = cx + DIRS[d][0];
                const ny = cy + DIRS[d][1];
                if (nx < 0 || ny < 0 || nx + w > BOARD_W || ny + h > BOARD_H) continue;
                const nf = ny * BOARD_W + nx;
                if (_ffVisited[nf]) continue;
                if (!footprintClear(packed, id, w, h, nx, ny)) continue;
                _ffVisited[nf] = 1;
                _ffQueue[tail++] = nf;
                moves.push({ pieceId: id, fromX: px, fromY: py, toX: nx, toY: ny });
            }
        }
    }
    return moves;
}

// move を packed に適用した新しい盤面を返す (元の packed は破壊しない)。
// 注意: clear ループと set ループは分離する必要がある。同 dy で clear→set を
// 交互に書くと、たとえば 1x2駒が縦に1マス動く時 (fromY..fromY+1 と toY..toY+1
// が overlap)、dy=1 の clear が dy=0 の set を上書きして壊れる。
function applyMoveToBoard(packed, move) {
    const out = new Uint8Array(16);
    out.set(packed);
    const w = PIECE_W[move.pieceId];
    const h = PIECE_H[move.pieceId];
    // Clear old footprint first
    for (let dy = 0; dy < h; dy++) {
        const fy = (move.fromY + dy) * BOARD_W;
        for (let dx = 0; dx < w; dx++) {
            setCell(out, fy + move.fromX + dx, 0);
        }
    }
    // Then set new footprint
    for (let dy = 0; dy < h; dy++) {
        const ty = (move.toY + dy) * BOARD_W;
        for (let dx = 0; dx < w; dx++) {
            setCell(out, ty + move.toX + dx, move.pieceId);
        }
    }
    return out;
}

// 娘 (id=1, 2x2) が (2, 3) で脱出位置にあるか。
function isGoalPacked(packed) {
    return getCell(packed, 3 * BOARD_W + 2) === DAUGHTER_ID
        && getCell(packed, 3 * BOARD_W + 3) === DAUGHTER_ID
        && getCell(packed, 4 * BOARD_W + 2) === DAUGHTER_ID
        && getCell(packed, 4 * BOARD_W + 3) === DAUGHTER_ID;
}

// ----- visited 集合: u32ペアのopen-addressing -----
class U64HashSet {
    constructor(expectedSize) {
        let cap = 16;
        while (cap < expectedSize * 2) cap <<= 1;
        this.cap = cap;
        this.mask = cap - 1;
        this.slots = new Uint32Array(cap * 2);
        this.size = 0;
        this.hasZero = false;
    }
    add(lo, hi) {
        if (lo === 0 && hi === 0) {
            if (this.hasZero) return false;
            this.hasZero = true; this.size++;
            return true;
        }
        let idx = lo & this.mask;
        const slots = this.slots;
        for (;;) {
            const baseI = idx << 1;
            const sLo = slots[baseI];
            const sHi = slots[baseI | 1];
            if (sLo === 0 && sHi === 0) {
                slots[baseI] = lo; slots[baseI | 1] = hi;
                this.size++;
                if (this.size * 2 > this.cap) this._grow();
                return true;
            }
            if (sLo === lo && sHi === hi) return false;
            idx = (idx + 1) & this.mask;
        }
    }
    _grow() {
        const oldSlots = this.slots;
        const oldCap = this.cap;
        this.cap = oldCap << 1;
        this.mask = this.cap - 1;
        this.slots = new Uint32Array(this.cap * 2);
        for (let i = 0; i < oldCap; i++) {
            const lo = oldSlots[i << 1];
            const hi = oldSlots[(i << 1) | 1];
            if (lo === 0 && hi === 0) continue;
            let idx = lo & this.mask;
            const slots = this.slots;
            for (;;) {
                const baseI = idx << 1;
                if (slots[baseI] === 0 && slots[baseI | 1] === 0) {
                    slots[baseI] = lo; slots[baseI | 1] = hi;
                    break;
                }
                idx = (idx + 1) & this.mask;
            }
        }
    }
}

// ----- BFSキュー: parent-pointer 復元、典型 ~40B/state -----
class BFSQueue {
    constructor(initialCap = 1024) {
        this.cap = initialCap;
        this.size = 0;
        this.boards  = new Uint8Array(this.cap * 16);
        this.hashes  = new Uint32Array(this.cap * 4);   // hLo, hHi, mLo, mHi
        this.parents = new Int32Array(this.cap);
        this.moves   = new Uint8Array(this.cap * 5);    // pieceId, fromX, fromY, toX, toY
        this.depths  = new Uint16Array(this.cap);
    }
    push(board, hashes, parentIdx, move, depth) {
        if (this.size >= this.cap) this._grow();
        const i = this.size++;
        this.boards.set(board, i * 16);
        const hi = i * 4;
        this.hashes[hi]     = hashes[0];
        this.hashes[hi + 1] = hashes[1];
        this.hashes[hi + 2] = hashes[2];
        this.hashes[hi + 3] = hashes[3];
        this.parents[i] = parentIdx;
        if (move) {
            const mi = i * 5;
            this.moves[mi]     = move.pieceId;
            this.moves[mi + 1] = move.fromX;
            this.moves[mi + 2] = move.fromY;
            this.moves[mi + 3] = move.toX;
            this.moves[mi + 4] = move.toY;
        }
        this.depths[i] = depth;
        return i;
    }
    _grow() {
        const newCap = this.cap * 2;
        const nb = new Uint8Array(newCap * 16);  nb.set(this.boards);  this.boards  = nb;
        const nh = new Uint32Array(newCap * 4);  nh.set(this.hashes);  this.hashes  = nh;
        const np = new Int32Array(newCap);       np.set(this.parents); this.parents = np;
        const nm = new Uint8Array(newCap * 5);   nm.set(this.moves);   this.moves   = nm;
        const nd = new Uint16Array(newCap);      nd.set(this.depths);  this.depths  = nd;
        this.cap = newCap;
    }
    boardAt(i) { return this.boards.subarray(i * 16, i * 16 + 16); }
    moveAt(i) {
        const mi = i * 5;
        return {
            pieceId: this.moves[mi],
            fromX:   this.moves[mi + 1],
            fromY:   this.moves[mi + 2],
            toX:     this.moves[mi + 3],
            toY:     this.moves[mi + 4],
        };
    }
}

// ----- PuzzleSolver: 既存game.jsのAPIに合わせる -----
class PuzzleSolver {
    constructor(game) {
        this.game = game;
        this.cancelled = false;
        // 既存game.jsはこれらを書き換えるが、新ソルバはBFS固定なのでmaxDepthは事実上ignore。
        this.maxDepth = 60;
        this.maxStates = 30_000_000;
        this.maxTime = 600_000;  // 10 min hard ceiling
    }
    cancel() { this.cancelled = true; }
    // 既存game.jsの requestSolve は solveAStar -> solveBidirectional の順に呼ぶ。
    // 我々のBFSは常に最短を返すので、最初の solveAStar で必ず result が返り、
    // フォールバックは起きない。互換のため両メソッドとも同じ実装を呼ぶ。
    async solveAStar()         { return this._solve(); }
    async solveBidirectional() { return this._solve(); }
    async solve()              { return this._solve(); }

    async _solve() {
        const start = Date.now();
        try {
            await loadSolverData();
        } catch (e) {
            // テーブル無しでも純BFSは動くが、初期盤面 (49手) は28M状態必要で実用不可。
            // ユーザーには警告を出して続行。
            this.game.updateSolverProgress(0, 0, 0, 0, `テーブル読込失敗: ${e.message} (純BFS)`);
            console.warn('Solver data load failed:', e);
        }

        const initialPacked = packFromPieces(this.game.pieces);
        if (isGoalPacked(initialPacked)) {
            return { moves: [], steps: 0, states: 1, time: Date.now() - start, algorithm: 'BFS+table' };
        }

        const hashScratch = new Uint32Array(4);
        computeHashes(initialPacked, hashScratch);
        const initCLo = canonicalLo(hashScratch);
        const initCHi = canonicalHi(hashScratch);

        // 初期状態がいきなりテーブル内なら、純粋にテーブル誘導で全経路が出る。
        const initD = tableLookup(initCLo, initCHi);
        if (initD >= 0) {
            const tail = this._walkTable(initialPacked, hashScratch, initD);
            return { moves: tail.moves, steps: tail.moves.length, states: tail.statesWalked, time: Date.now() - start, algorithm: 'table-only' };
        }

        const visited = new U64HashSet(1 << 20);  // 1M, 自動拡張
        visited.add(initCLo, initCHi);

        const queue = new BFSQueue(1024);
        queue.push(initialPacked, hashScratch, -1, null, 0);

        const posBuf = new Int32Array(MAX_PIECE_ID + 1);
        const newHashes = new Uint32Array(4);

        let head = 0;
        let lastUI = start;
        let yieldsSince = 0;
        const UI_INTERVAL_MS = 250;
        const YIELD_INTERVAL_STATES = 50_000;

        this.game.updateSolverProgress(0, 0, 1, 0, 'BFS探索開始...');

        while (head < queue.size) {
            if (this.cancelled) throw new Error('cancelled');

            const i = head++;
            const board = queue.boardAt(i);
            const baseHi = i * 4;
            hashScratch[0] = queue.hashes[baseHi];
            hashScratch[1] = queue.hashes[baseHi + 1];
            hashScratch[2] = queue.hashes[baseHi + 2];
            hashScratch[3] = queue.hashes[baseHi + 3];
            const depth = queue.depths[i];

            yieldsSince++;
            const now = Date.now();
            if (now - lastUI > UI_INTERVAL_MS) {
                const elapsedSec = (now - start) / 1000;
                const rate = elapsedSec > 0 ? Math.round(head / elapsedSec) : 0;
                const msg = `深度${depth} | ${rate.toLocaleString()}状態/秒`;
                this.game.updateSolverProgress(head, depth, queue.size - head, Math.min(depth * 2, 99), msg);
                this.game.updateStatus(`探索中... ${head.toLocaleString()}状態 (${elapsedSec.toFixed(1)}秒)`);
                lastUI = now;
            }
            if (yieldsSince >= YIELD_INTERVAL_STATES) {
                await new Promise(r => setTimeout(r, 0));
                yieldsSince = 0;
            }
            if (now - start > this.maxTime) throw new Error(`時間制限: ${Math.round((now - start) / 1000)}秒`);
            if (head > this.maxStates) throw new Error(`状態数制限: ${this.maxStates.toLocaleString()}状態`);

            // ゴール直接ヒット (基本的にtable hit で吸収されるが、テーブル無効時のフォールバック)
            if (isGoalPacked(board)) {
                const path = this._reconstructPath(queue, i);
                return { moves: path, steps: path.length, states: head, time: Date.now() - start, algorithm: 'BFS' };
            }

            const moves = generateMoves(board, posBuf);
            for (let mi = 0; mi < moves.length; mi++) {
                const mv = moves[mi];
                applyMoveHashes(hashScratch, newHashes, mv.pieceId, mv.fromX, mv.fromY, mv.toX, mv.toY);
                const cLo = canonicalLo(newHashes);
                const cHi = canonicalHi(newHashes);
                if (!visited.add(cLo, cHi)) continue;

                const newBoard = applyMoveToBoard(board, mv);

                // ゴール即達 (mirror-canonicalで一度dedupされた状態でも子で発生しうる)
                if (isGoalPacked(newBoard)) {
                    const path = this._reconstructPath(queue, i);
                    path.push(mv);
                    return { moves: path, steps: path.length, states: head, time: Date.now() - start, algorithm: 'BFS' };
                }

                // テーブルヒット: 残りはテーブル誘導で最適に歩く
                const tableD = tableLookup(cLo, cHi);
                if (tableD >= 0) {
                    const bfsPath = this._reconstructPath(queue, i);
                    bfsPath.push(mv);
                    const tail = this._walkTable(newBoard, newHashes, tableD);
                    bfsPath.push(...tail.moves);
                    return {
                        moves: bfsPath,
                        steps: bfsPath.length,
                        states: head + tail.statesWalked,
                        time: Date.now() - start,
                        algorithm: 'BFS+table'
                    };
                }

                queue.push(newBoard, newHashes, i, mv, depth + 1);
            }
        }

        if (this.cancelled) throw new Error('cancelled');
        return null;
    }

    _reconstructPath(queue, idx) {
        const path = [];
        let i = idx;
        while (queue.parents[i] >= 0) {
            path.push(queue.moveAt(i));
            i = queue.parents[i];
        }
        path.reverse();
        return path;
    }

    // 与えられた状態 (packed, hashes) からテーブルを使って G_final まで貪欲に降りる。
    // 各ステップで「次状態の d_table が現在の d_table - 1」となる移動を選択。
    // ハッシュ衝突等で見つからない場合は例外を投げる (堅牢性チェック)。
    _walkTable(packed, hashes, currentD) {
        const moves = [];
        const posBuf = new Int32Array(MAX_PIECE_ID + 1);
        const nh = new Uint32Array(4);
        let statesWalked = 1;
        while (currentD > 0) {
            const candidates = generateMoves(packed, posBuf);
            let picked = null;
            let pickedBoard = null;
            let pickedHashes = null;
            for (let i = 0; i < candidates.length; i++) {
                const mv = candidates[i];
                applyMoveHashes(hashes, nh, mv.pieceId, mv.fromX, mv.fromY, mv.toX, mv.toY);
                const cLo = canonicalLo(nh);
                const cHi = canonicalHi(nh);
                const d = tableLookup(cLo, cHi);
                if (d === currentD - 1) {
                    picked = mv;
                    pickedBoard = applyMoveToBoard(packed, mv);
                    pickedHashes = new Uint32Array(nh);  // snapshot
                    break;
                }
            }
            if (!picked) {
                throw new Error(`table walk stuck at d=${currentD}`);
            }
            moves.push(picked);
            packed = pickedBoard;
            hashes[0] = pickedHashes[0]; hashes[1] = pickedHashes[1];
            hashes[2] = pickedHashes[2]; hashes[3] = pickedHashes[3];
            currentD--;
            statesWalked++;
        }
        return { moves, statesWalked };
    }
}

// グローバル公開 (game.js から `new PuzzleSolver(this)` で参照される)
window.PuzzleSolver = PuzzleSolver;

// デバッグ用エクスポート (window.testSolver から触れる)
window.__solverInternal = { loadSolverData, packFromPieces, computeHashes, generateMoves, isGoalPacked, tableLookup };

})();
