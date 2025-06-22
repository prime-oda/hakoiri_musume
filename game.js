// 箱入り娘の大家族 - HTML5 Canvas版

class Board {
    constructor(width, height, cellSize = 60, margin = 50) {
        this.width = width;
        this.height = height;
        this.cellSize = cellSize;
        this.margin = margin;
        this.boardWidth = this.width * this.cellSize;
        this.boardHeight = this.height * this.cellSize;
        this.startX = this.margin;
        this.startY = this.margin;
        this.grid = Array(this.height).fill().map(() => Array(this.width).fill(0));
    }

    getCellRect(x, y) {
        const rectX = this.startX + x * this.cellSize;
        const rectY = this.startY + y * this.cellSize;
        return { x: rectX, y: rectY, width: this.cellSize, height: this.cellSize };
    }

    gridToPixel(gridX, gridY) {
        const pixelX = this.startX + gridX * this.cellSize;
        const pixelY = this.startY + gridY * this.cellSize;
        return { x: pixelX, y: pixelY };
    }

    pixelToGrid(pixelX, pixelY) {
        if (pixelX < this.startX || pixelX >= this.startX + this.boardWidth ||
            pixelY < this.startY || pixelY >= this.startY + this.boardHeight) {
            return { x: null, y: null };
        }
        
        const gridX = Math.floor((pixelX - this.startX) / this.cellSize);
        const gridY = Math.floor((pixelY - this.startY) / this.cellSize);
        return { x: gridX, y: gridY };
    }

    isValidPosition(x, y) {
        return x >= 0 && x < this.width && y >= 0 && y < this.height;
    }

    isEmpty(x, y) {
        if (!this.isValidPosition(x, y)) return false;
        return this.grid[y][x] === 0;
    }

    draw(ctx) {
        // 盤面の背景
        ctx.fillStyle = '#e9ecef';
        ctx.fillRect(this.startX, this.startY, this.boardWidth, this.boardHeight);
        
        // 脱出口を描画（下端の中央2×2マス）
        this.drawExitArea(ctx);
        
        // 盤面の枠線
        ctx.strokeStyle = '#333';
        ctx.lineWidth = 2;
        ctx.strokeRect(this.startX, this.startY, this.boardWidth, this.boardHeight);

        // グリッド線
        ctx.strokeStyle = '#333';
        ctx.lineWidth = 1;
        
        // 縦線
        for (let x = 0; x <= this.width; x++) {
            const lineX = this.startX + x * this.cellSize;
            ctx.beginPath();
            ctx.moveTo(lineX, this.startY);
            ctx.lineTo(lineX, this.startY + this.boardHeight);
            ctx.stroke();
        }
        
        // 横線
        for (let y = 0; y <= this.height; y++) {
            const lineY = this.startY + y * this.cellSize;
            ctx.beginPath();
            ctx.moveTo(this.startX, lineY);
            ctx.lineTo(this.startX + this.boardWidth, lineY);
            ctx.stroke();
        }
    }

    drawExitArea(ctx) {
        // 脱出口（下端の中央2×2マス）
        const exitX = 2;
        const exitY = 3;
        const exitWidth = 2;
        const exitHeight = 2;
        
        const pos = this.gridToPixel(exitX, exitY);
        const pixelWidth = exitWidth * this.cellSize;
        const pixelHeight = exitHeight * this.cellSize;
        
        // 脱出エリアの背景（薄い緑色）
        ctx.fillStyle = 'rgba(144, 238, 144, 0.3)';
        ctx.fillRect(pos.x, pos.y, pixelWidth, pixelHeight);
        
        // 脱出エリアの枠線（緑色）
        ctx.strokeStyle = '#32CD32';
        ctx.lineWidth = 3;
        ctx.setLineDash([5, 5]);
        ctx.strokeRect(pos.x, pos.y, pixelWidth, pixelHeight);
        ctx.setLineDash([]);
        
        // 「脱出口」のテキスト
        ctx.fillStyle = '#228B22';
        ctx.font = 'bold 14px "Noto Sans JP", sans-serif';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        
        const textX = pos.x + pixelWidth / 2;
        const textY = pos.y + pixelHeight / 2;
        
        // テキストの背景
        ctx.fillStyle = 'rgba(255, 255, 255, 0.8)';
        ctx.fillRect(textX - 30, textY - 8, 60, 16);
        
        // テキスト
        ctx.fillStyle = '#228B22';
        ctx.fillText('脱出口', textX, textY);
        
        // 線の設定をリセット
        ctx.lineWidth = 1;
    }
}

class Piece {
    constructor(id, x, y, width, height, color, name) {
        this.id = id;
        this.x = x;
        this.y = y;
        this.width = width;
        this.height = height;
        this.color = color;
        this.name = name;
        this.selected = false;
    }

    getOccupiedCells() {
        const cells = [];
        for (let dy = 0; dy < this.height; dy++) {
            for (let dx = 0; dx < this.width; dx++) {
                cells.push({ x: this.x + dx, y: this.y + dy });
            }
        }
        return cells;
    }

    canSlideTo(newX, newY, board) {
        // 移動先のセルが空いているかチェック
        for (let dy = 0; dy < this.height; dy++) {
            for (let dx = 0; dx < this.width; dx++) {
                const checkX = newX + dx;
                const checkY = newY + dy;
                
                // 自分の現在位置は除外してチェック
                const currentCells = this.getOccupiedCells();
                const isCurrentPosition = currentCells.some(cell => 
                    cell.x === checkX && cell.y === checkY
                );
                
                if (!isCurrentPosition && !board.isEmpty(checkX, checkY)) {
                    return false;
                }
            }
        }
        return true;
    }

    canMoveTo(newX, newY, board) {
        // 盤面の境界チェック
        if (newX < 0 || newY < 0 || 
            newX + this.width > board.width || 
            newY + this.height > board.height) {
            return false;
        }

        // 現在位置から新しい位置への移動がスライド可能かチェック
        const dx = newX - this.x;
        const dy = newY - this.y;

        // 移動なしの場合は無効
        if (dx === 0 && dy === 0) return false;

        // 斜め移動は無効（上下左右のみ）
        if (dx !== 0 && dy !== 0) return false;

        // 段階的にチェック
        const steps = Math.max(Math.abs(dx), Math.abs(dy));
        const stepX = dx === 0 ? 0 : (dx > 0 ? 1 : -1);
        const stepY = dy === 0 ? 0 : (dy > 0 ? 1 : -1);

        for (let step = 1; step <= steps; step++) {
            const intermediateX = this.x + stepX * step;
            const intermediateY = this.y + stepY * step;
            
            if (!this.canSlideTo(intermediateX, intermediateY, board)) {
                return false;
            }
        }
        return true;
    }

    getPossibleMoves(board) {
        const possibleMoves = [];
        const directions = [
            { x: 0, y: -1 }, // 上
            { x: 0, y: 1 },  // 下
            { x: -1, y: 0 }, // 左
            { x: 1, y: 0 }   // 右
        ];

        for (const dir of directions) {
            let step = 1;
            while (true) {
                const newX = this.x + dir.x * step;
                const newY = this.y + dir.y * step;

                // 境界チェック
                if (newX < 0 || newY < 0 || 
                    newX + this.width > board.width || 
                    newY + this.height > board.height) {
                    break;
                }

                if (this.canSlideTo(newX, newY, board)) {
                    possibleMoves.push({ x: newX, y: newY });
                    step++;
                } else {
                    break;
                }
            }
        }
        return possibleMoves;
    }

    moveTo(newX, newY, board) {
        if (this.canMoveTo(newX, newY, board)) {
            // 現在の位置をクリア
            const currentCells = this.getOccupiedCells();
            for (const cell of currentCells) {
                board.grid[cell.y][cell.x] = 0;
            }

            // 新しい位置を設定
            this.x = newX;
            this.y = newY;

            // 新しい位置にIDを設定
            const newCells = this.getOccupiedCells();
            for (const cell of newCells) {
                board.grid[cell.y][cell.x] = this.id;
            }
            return true;
        }
        return false;
    }

    draw(ctx, board) {
        const pos = board.gridToPixel(this.x, this.y);
        const widthPixels = this.width * board.cellSize;
        const heightPixels = this.height * board.cellSize;

        // 駒の本体を描画
        ctx.fillStyle = this.color;
        ctx.fillRect(pos.x + 2, pos.y + 2, widthPixels - 4, heightPixels - 4);

        // 選択状態の場合は赤い枠線
        if (this.selected) {
            ctx.strokeStyle = '#ff0000';
            ctx.lineWidth = 3;
        } else {
            ctx.strokeStyle = '#000000';
            ctx.lineWidth = 2;
        }
        ctx.strokeRect(pos.x + 2, pos.y + 2, widthPixels - 4, heightPixels - 4);

        // 駒の名前を描画
        if (this.name) {
            ctx.fillStyle = '#ffffff';
            ctx.font = 'bold 16px "Noto Sans JP", sans-serif';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            
            const textX = pos.x + widthPixels / 2;
            const textY = pos.y + heightPixels / 2;
            
            // テキストの影
            ctx.fillStyle = '#000000';
            ctx.fillText(this.name, textX + 1, textY + 1);
            
            // テキスト本体
            ctx.fillStyle = '#ffffff';
            ctx.fillText(this.name, textX, textY);
        }
    }
}

// 駒の種類を定義
class PieceType {
    static COLORS = {
        DAUGHTER: 'rgb(255, 100, 100)',
        FATHER: 'rgb(100, 50, 200)',
        MOTHER: 'rgb(200, 50, 100)',
        HEAD_CLERK: 'rgb(50, 150, 200)',
        ASSISTANT: 'rgb(255, 150, 0)',
        MAID: 'rgb(100, 200, 50)',
        CLERK: 'rgb(200, 150, 50)',
        GRANDFATHER: 'rgb(150, 100, 50)',
        GRANDMOTHER: 'rgb(200, 100, 200)',
        DOG: 'rgb(100, 100, 100)',
        SISTER_IN_LAW: 'rgb(150, 200, 150)',
        APPRENTICE: 'rgb(200, 200, 100)'
    };

    static createDaughter(id, x, y) {
        return new Piece(id, x, y, 2, 2, this.COLORS.DAUGHTER, '娘');
    }

    static createFather(id, x, y) {
        return new Piece(id, x, y, 1, 2, this.COLORS.FATHER, '父');
    }

    static createMother(id, x, y) {
        return new Piece(id, x, y, 1, 2, this.COLORS.MOTHER, '母');
    }

    static createHeadClerk(id, x, y) {
        return new Piece(id, x, y, 4, 1, this.COLORS.HEAD_CLERK, '大番頭');
    }

    static createAssistant(id, x, y) {
        return new Piece(id, x, y, 1, 1, this.COLORS.ASSISTANT, '手代');
    }

    static createMaid(id, x, y) {
        return new Piece(id, x, y, 2, 1, this.COLORS.MAID, '女中');
    }

    static createClerk(id, x, y) {
        return new Piece(id, x, y, 2, 1, this.COLORS.CLERK, '番頭');
    }

    static createGrandfather(id, x, y) {
        return new Piece(id, x, y, 2, 1, this.COLORS.GRANDFATHER, '祖父');
    }

    static createGrandmother(id, x, y) {
        return new Piece(id, x, y, 2, 1, this.COLORS.GRANDMOTHER, '祖母');
    }

    static createDog(id, x, y) {
        return new Piece(id, x, y, 1, 1, this.COLORS.DOG, '犬');
    }

    static createSisterInLaw(id, x, y) {
        return new Piece(id, x, y, 1, 1, this.COLORS.SISTER_IN_LAW, '嫁');
    }

    static createApprentice(id, x, y, number) {
        return new Piece(id, x, y, 1, 1, this.COLORS.APPRENTICE, `丁${number}`);
    }
}

class Game {
    constructor(canvasId) {
        this.canvas = document.getElementById(canvasId);
        this.ctx = this.canvas.getContext('2d');
        this.board = new Board(6, 5);
        this.pieces = [];
        this.selectedPiece = null;
        this.dragging = false;
        this.dragOffset = { x: 0, y: 0 };
        this.possibleMoves = [];
        this.hoverMove = null;
        this.moveCount = 0;
        this.gameCompleted = false;

        this.setupInitialPieces();
        this.setupEventListeners();
        this.setupUI();
        this.draw();
    }

    setupInitialPieces() {
        let pieceId = 1;

        // 娘（2×2）- 中央（脱出目標駒）
        this.pieces.push(PieceType.createDaughter(pieceId++, 2, 0));

        // 父（1×2）- 左
        this.pieces.push(PieceType.createFather(pieceId++, 1, 0));

        // 母（1×2）- 右
        this.pieces.push(PieceType.createMother(pieceId++, 4, 0));

        // 手代（1×1）- 左下
        this.pieces.push(PieceType.createAssistant(pieceId++, 0, 2));

        // 大番頭（4×1）- 横
        this.pieces.push(PieceType.createHeadClerk(pieceId++, 1, 2));

        // 兄嫁（1×1）- 右下
        this.pieces.push(PieceType.createSisterInLaw(pieceId++, 5, 2));

        // 丁稚1（1×1）- 左
        this.pieces.push(PieceType.createApprentice(pieceId++, 0, 3, 1));

        // 番頭（2×1）- 横向き
        this.pieces.push(PieceType.createClerk(pieceId++, 1, 3));

        // 女中（2×1）- 横向き
        this.pieces.push(PieceType.createMaid(pieceId++, 3, 3));

        // 丁稚2（1×1）- 右
        this.pieces.push(PieceType.createApprentice(pieceId++, 5, 3, 2));

        // 番犬（1×1）- 左下
        this.pieces.push(PieceType.createDog(pieceId++, 0, 4));

        // 祖父（2×1）- 横向き
        this.pieces.push(PieceType.createGrandfather(pieceId++, 1, 4));

        // 祖母（2×1）- 横向き
        this.pieces.push(PieceType.createGrandmother(pieceId++, 3, 4));

        // 丁稚3（1×1）- 右下
        this.pieces.push(PieceType.createApprentice(pieceId++, 5, 4, 3));

        // 盤面に駒を配置
        for (const piece of this.pieces) {
            const cells = piece.getOccupiedCells();
            for (const cell of cells) {
                this.board.grid[cell.y][cell.x] = piece.id;
            }
        }
    }

    setupEventListeners() {
        // マウスイベント
        this.canvas.addEventListener('mousedown', (e) => this.handlePointerDown(e));
        this.canvas.addEventListener('mousemove', (e) => this.handlePointerMove(e));
        this.canvas.addEventListener('mouseup', (e) => this.handlePointerUp(e));
        this.canvas.addEventListener('mouseleave', (e) => this.handlePointerUp(e));
        
        // タッチイベント
        this.canvas.addEventListener('touchstart', (e) => {
            e.preventDefault(); // スクロール等を防止
            this.handlePointerDown(e);
        }, { passive: false });
        
        this.canvas.addEventListener('touchmove', (e) => {
            e.preventDefault(); // スクロール等を防止
            this.handlePointerMove(e);
        }, { passive: false });
        
        this.canvas.addEventListener('touchend', (e) => {
            e.preventDefault();
            this.handlePointerUp(e);
        }, { passive: false });
        
        this.canvas.addEventListener('touchcancel', (e) => {
            e.preventDefault();
            this.handlePointerUp(e);
        }, { passive: false });
    }

    setupUI() {
        // リセットボタン
        document.getElementById('resetBtn').addEventListener('click', () => {
            this.resetGame();
        });

        // ヘルプボタン
        document.getElementById('helpBtn').addEventListener('click', () => {
            document.getElementById('helpModal').style.display = 'block';
        });

        // モーダルを閉じる
        document.querySelector('.close').addEventListener('click', () => {
            document.getElementById('helpModal').style.display = 'none';
        });

        // モーダル外をクリックして閉じる
        window.addEventListener('click', (e) => {
            const modal = document.getElementById('helpModal');
            if (e.target === modal) {
                modal.style.display = 'none';
            }
        });
    }

    getPointerPos(e) {
        const rect = this.canvas.getBoundingClientRect();
        const scaleX = this.canvas.width / rect.width;
        const scaleY = this.canvas.height / rect.height;
        
        let clientX, clientY;
        
        if (e.touches && e.touches.length > 0) {
            // タッチイベントの場合
            clientX = e.touches[0].clientX;
            clientY = e.touches[0].clientY;
        } else if (e.changedTouches && e.changedTouches.length > 0) {
            // touchendイベントの場合
            clientX = e.changedTouches[0].clientX;
            clientY = e.changedTouches[0].clientY;
        } else {
            // マウスイベントの場合
            clientX = e.clientX;
            clientY = e.clientY;
        }
        
        return {
            x: (clientX - rect.left) * scaleX,
            y: (clientY - rect.top) * scaleY
        };
    }

    getMousePos(e) {
        // 後方互換性のため
        return this.getPointerPos(e);
    }

    handleMouseDown(e) {
        // 後方互換性のため
        return this.handlePointerDown(e);
    }

    handleMouseMove(e) {
        // 後方互換性のため
        return this.handlePointerMove(e);
    }

    handleMouseUp(e) {
        // 後方互換性のため
        return this.handlePointerUp(e);
    }

    getPieceAtPosition(pos) {
        const gridPos = this.board.pixelToGrid(pos.x, pos.y);
        
        if (gridPos.x === null || gridPos.y === null) return null;

        // 駒を逆順でチェック（後から描画された駒を優先）
        for (let i = this.pieces.length - 1; i >= 0; i--) {
            const piece = this.pieces[i];
            const cells = piece.getOccupiedCells();
            if (cells.some(cell => cell.x === gridPos.x && cell.y === gridPos.y)) {
                return piece;
            }
        }
        return null;
    }

    handlePointerDown(e) {
        if (this.gameCompleted) return;
        
        const pointerPos = this.getPointerPos(e);
        const piece = this.getPieceAtPosition(pointerPos);
        
        if (piece) {
            // 前の選択を解除
            if (this.selectedPiece) {
                this.selectedPiece.selected = false;
            }

            // 可能な移動先を取得
            this.possibleMoves = piece.getPossibleMoves(this.board);

            // 移動先が1つだけの場合は自動で移動
            if (this.possibleMoves.length === 1) {
                const move = this.possibleMoves[0];
                if (piece.moveTo(move.x, move.y, this.board)) {
                    this.moveCount++;
                    this.updateStatus(`${piece.name}を自動で移動しました`);
                    this.updateUI();
                }
                this.selectedPiece = null;
                this.possibleMoves = [];
                this.draw();
                return;
            }

            // 移動先が0個または2個以上の場合は通常のドラッグモードに
            this.selectedPiece = piece;
            piece.selected = true;
            this.dragging = true;

            // ドラッグオフセットを計算
            const piecePos = this.board.gridToPixel(piece.x, piece.y);
            this.dragOffset.x = pointerPos.x - piecePos.x;
            this.dragOffset.y = pointerPos.y - piecePos.y;

            // ステータス更新
            if (this.possibleMoves.length === 0) {
                this.updateStatus(`${piece.name}は移動できません`);
            } else {
                this.updateStatus(`${piece.name}をドラッグして移動してください`);
            }

            this.draw();
        }
    }

    handlePointerMove(e) {
        if (this.gameCompleted) return;
        
        if (this.selectedPiece && this.dragging) {
            const pointerPos = this.getPointerPos(e);
            
            // ドロップ位置を計算
            const dropPixelX = pointerPos.x - this.dragOffset.x;
            const dropPixelY = pointerPos.y - this.dragOffset.y;

            // ピクセル座標をグリッド座標に変換
            const dropGridX = Math.round((dropPixelX - this.board.startX) / this.board.cellSize);
            const dropGridY = Math.round((dropPixelY - this.board.startY) / this.board.cellSize);

            // ドラッグ位置が可能な移動先の中にあるかチェック
            this.hoverMove = null;
            for (const move of this.possibleMoves) {
                if (move.x === dropGridX && move.y === dropGridY) {
                    this.hoverMove = move;
                    break;
                }
            }

            this.draw();
        }
    }

    handlePointerUp(e) {
        if (this.dragging && this.selectedPiece) {
            // ハイライトされた移動先がある場合はそこに移動
            if (this.hoverMove) {
                if (this.selectedPiece.moveTo(this.hoverMove.x, this.hoverMove.y, this.board)) {
                    this.moveCount++;
                    this.updateStatus(`${this.selectedPiece.name}を移動しました`);
                    this.updateUI();
                } else {
                    this.updateStatus('移動に失敗しました');
                }
            } else {
                this.updateStatus('有効な移動先にドロップしてください');
            }
        }

        // ドラッグ状態をリセット
        this.dragging = false;
        if (this.selectedPiece) {
            this.selectedPiece.selected = false;
            this.selectedPiece = null;
        }
        this.possibleMoves = [];
        this.hoverMove = null;
        this.draw();
    }

    drawMoveOverlays() {
        if (this.possibleMoves.length === 0 || !this.selectedPiece) return;

        for (const move of this.possibleMoves) {
            // 駒のサイズ分のオーバーレイを描画
            for (let dy = 0; dy < this.selectedPiece.height; dy++) {
                for (let dx = 0; dx < this.selectedPiece.width; dx++) {
                    const cellX = move.x + dx;
                    const cellY = move.y + dy;
                    
                    if (this.board.isValidPosition(cellX, cellY)) {
                        const pos = this.board.gridToPixel(cellX, cellY);
                        
                        // ハイライト色か通常のオーバーレイ色かを決定
                        if (this.hoverMove && 
                            this.hoverMove.x === move.x && this.hoverMove.y === move.y) {
                            this.ctx.fillStyle = 'rgba(255, 200, 0, 0.6)'; // 黄色ハイライト
                        } else {
                            this.ctx.fillStyle = 'rgba(100, 150, 255, 0.3)'; // 青色オーバーレイ
                        }
                        
                        this.ctx.fillRect(pos.x, pos.y, this.board.cellSize, this.board.cellSize);
                    }
                }
            }
        }
    }

    draw() {
        // キャンバスをクリア
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);

        // 盤面を描画
        this.board.draw(this.ctx);

        // 移動可能マスのオーバーレイを描画
        this.drawMoveOverlays();

        // 駒を描画
        for (const piece of this.pieces) {
            piece.draw(this.ctx, this.board);
        }
    }

    updateStatus(message) {
        document.getElementById('statusMessage').textContent = message;
    }

    updateUI() {
        document.getElementById('moveCount').textContent = this.moveCount;
        
        // 勝利条件をチェック（娘が脱出位置に到達）
        this.checkWinCondition();
    }

    checkWinCondition() {
        const daughter = this.pieces.find(p => p.name === '娘');
        if (!daughter) return;

        // 娘が脱出位置（下端の中央2マス）に到達したかチェック
        // 脱出位置: (2,3) から (3,4) の2×2エリア
        const exitX = 2;
        const exitY = 3;
        
        if (daughter.x === exitX && daughter.y === exitY) {
            this.gameCompleted = true;
            this.updateStatus('🎉 おめでとうございます！娘を脱出させました！');
            this.showWinModal();
        }
    }

    showWinModal() {
        // 勝利メッセージを表示
        setTimeout(() => {
            const winMessage = `ゲームクリア！\n\n手数: ${this.moveCount}手\n\nもう一度プレイしますか？`;
            if (confirm(winMessage)) {
                this.resetGame();
            }
        }, 500);
    }

    resetGame() {
        // ゲームをリセット
        this.pieces = [];
        this.selectedPiece = null;
        this.dragging = false;
        this.possibleMoves = [];
        this.hoverMove = null;
        this.moveCount = 0;
        this.gameCompleted = false;
        this.board.grid = Array(this.board.height).fill().map(() => Array(this.board.width).fill(0));
        
        this.setupInitialPieces();
        this.updateUI();
        this.updateStatus('ゲームをリセットしました');
        this.draw();
    }
}

// ゲーム開始
document.addEventListener('DOMContentLoaded', () => {
    new Game('gameCanvas');
});