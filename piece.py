import pygame

class Piece:
    def __init__(self, piece_id, x, y, width, height, color, name=""):
        self.id = piece_id
        self.x = x  # グリッド座標
        self.y = y  # グリッド座標
        self.width = width  # 駒の幅（マス数）
        self.height = height  # 駒の高さ（マス数）
        self.color = color
        self.name = name
        self.selected = False
        
    def get_occupied_cells(self):
        """この駒が占めるセルのリストを返す"""
        cells = []
        for dy in range(self.height):
            for dx in range(self.width):
                cells.append((self.x + dx, self.y + dy))
        return cells
        
    def can_move_to(self, new_x, new_y, board):
        """指定された位置に移動できるかチェック（スライドパズルルール）"""
        # 盤面の境界チェック
        if (new_x < 0 or new_y < 0 or 
            new_x + self.width > board.width or 
            new_y + self.height > board.height):
            return False
        
        # 現在位置から新しい位置への移動がスライド可能かチェック
        dx = new_x - self.x
        dy = new_y - self.y
        
        # 移動なしの場合は無効
        if dx == 0 and dy == 0:
            return False
        
        # 斜め移動は無効（上下左右のみ）
        if dx != 0 and dy != 0:
            return False
            
        # 1マス以上離れた位置への移動は段階的にチェック
        steps = max(abs(dx), abs(dy))
        step_x = 0 if dx == 0 else (1 if dx > 0 else -1)
        step_y = 0 if dy == 0 else (1 if dy > 0 else -1)
        
        # 各ステップで移動経路が空いているかチェック
        for step in range(1, steps + 1):
            intermediate_x = self.x + step_x * step
            intermediate_y = self.y + step_y * step
            
            # この中間位置に移動できるかチェック
            if not self._can_slide_to_position(intermediate_x, intermediate_y, board):
                return False
                
        return True
        
    def _can_slide_to_position(self, new_x, new_y, board):
        """指定された位置に直接スライドできるかチェック"""
        # 移動先のセルが空いているかチェック
        for dy in range(self.height):
            for dx in range(self.width):
                check_x = new_x + dx
                check_y = new_y + dy
                # 自分の現在位置は除外してチェック
                if (check_x, check_y) not in self.get_occupied_cells():
                    if not board.is_empty(check_x, check_y):
                        return False
        return True
    
    def get_possible_moves(self, board):
        """この駒が移動可能な位置のリストを返す"""
        possible_moves = []
        
        # 上下左右の全方向をチェック
        directions = [(0, -1), (0, 1), (-1, 0), (1, 0)]  # 上、下、左、右
        
        for dx, dy in directions:
            # 各方向に1マスずつ移動を試す
            step = 1
            while True:
                new_x = self.x + dx * step
                new_y = self.y + dy * step
                
                # 境界チェック
                if (new_x < 0 or new_y < 0 or 
                    new_x + self.width > board.width or 
                    new_y + self.height > board.height):
                    break
                
                # この位置に移動可能かチェック
                if self._can_slide_to_position(new_x, new_y, board):
                    possible_moves.append((new_x, new_y))
                    step += 1
                else:
                    # この方向にはもう移動できない
                    break
                    
        return possible_moves
        
    def move_to(self, new_x, new_y, board):
        """指定された位置に移動"""
        if self.can_move_to(new_x, new_y, board):
            # 現在の位置をクリア
            for cell_x, cell_y in self.get_occupied_cells():
                board.grid[cell_y][cell_x] = 0
                
            # 新しい位置を設定
            self.x = new_x
            self.y = new_y
            
            # 新しい位置にIDを設定
            for cell_x, cell_y in self.get_occupied_cells():
                board.grid[cell_y][cell_x] = self.id
            return True
        return False
        
    def draw(self, screen, board):
        """駒を描画"""
        pixel_x, pixel_y = board.grid_to_pixel(self.x, self.y)
        width_pixels = self.width * board.cell_size
        height_pixels = self.height * board.cell_size
        
        # 駒の本体を描画
        rect = pygame.Rect(pixel_x + 2, pixel_y + 2, 
                          width_pixels - 4, height_pixels - 4)
        pygame.draw.rect(screen, self.color, rect)
        
        # 選択状態の場合は枠線を描画
        if self.selected:
            pygame.draw.rect(screen, (255, 0, 0), rect, 3)
        else:
            pygame.draw.rect(screen, (0, 0, 0), rect, 2)
            
        # 駒の名前を描画（日本語フォント対応）
        if self.name:
            try:
                # macOSで利用可能な日本語フォントを試す
                font_paths = [
                    "/System/Library/Fonts/Hiragino Sans GB.ttc",
                    "/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc",
                    "/Library/Fonts/Arial Unicode MS.ttf",
                    None  # デフォルトフォント
                ]
                
                font = None
                for font_path in font_paths:
                    try:
                        if font_path:
                            font = pygame.font.Font(font_path, 20)
                        else:
                            font = pygame.font.Font(None, 24)
                        break
                    except:
                        continue
                
                if font:
                    text = font.render(self.name, True, (255, 255, 255))
                    text_rect = text.get_rect(center=rect.center)
                    screen.blit(text, text_rect)
            except:
                # フォント描画に失敗した場合は何もしない
                pass

# 駒の種類を定義
class PieceType:
    # 色の定義
    DAUGHTER_COLOR = (255, 100, 100)    # 赤（娘 - 目標駒）
    FATHER_COLOR = (100, 50, 200)       # 紫（父）
    MOTHER_COLOR = (200, 50, 100)       # ピンク（母）
    HEAD_CLERK_COLOR = (50, 150, 200)   # 青（大番頭）
    MAID_COLOR = (100, 200, 50)         # 緑（女中）
    CLERK_COLOR = (200, 150, 50)        # 黄色（番頭）
    GRANDFATHER_COLOR = (150, 100, 50)  # 茶色（祖父）
    GRANDMOTHER_COLOR = (200, 100, 200) # 紫ピンク（祖母）
    ASSISTANT_COLOR = (255, 150, 0)     # オレンジ（手代）
    DOG_COLOR = (100, 100, 100)         # グレー（番犬）
    SISTER_IN_LAW_COLOR = (150, 200, 150) # 薄緑（兄嫁）
    APPRENTICE_COLOR = (200, 200, 100)  # 薄黄色（丁稚）
    
    @staticmethod
    def create_daughter(piece_id, x, y):
        """娘（2×2）- 脱出させる目標の駒"""
        return Piece(piece_id, x, y, 2, 2, PieceType.DAUGHTER_COLOR, "娘")
        
    @staticmethod
    def create_father(piece_id, x, y):
        """父（1×2）- 縦長の駒"""
        return Piece(piece_id, x, y, 1, 2, PieceType.FATHER_COLOR, "父")
        
    @staticmethod
    def create_mother(piece_id, x, y):
        """母（1×2）- 縦長の駒"""
        return Piece(piece_id, x, y, 1, 2, PieceType.MOTHER_COLOR, "母")
        
    @staticmethod
    def create_head_clerk(piece_id, x, y):
        """大番頭（4×1）- 横長の大きな駒"""
        return Piece(piece_id, x, y, 4, 1, PieceType.HEAD_CLERK_COLOR, "大番頭")
        
    @staticmethod
    def create_maid(piece_id, x, y):
        """女中（2×1）- 横長の駒"""
        return Piece(piece_id, x, y, 2, 1, PieceType.MAID_COLOR, "女中")
        
    @staticmethod
    def create_clerk(piece_id, x, y):
        """番頭（2×1）- 横長の駒"""
        return Piece(piece_id, x, y, 2, 1, PieceType.CLERK_COLOR, "番頭")
        
    @staticmethod
    def create_grandfather(piece_id, x, y):
        """祖父（2×1）- 横長の駒"""
        return Piece(piece_id, x, y, 2, 1, PieceType.GRANDFATHER_COLOR, "祖父")
        
    @staticmethod
    def create_grandmother(piece_id, x, y):
        """祖母（2×1）- 横長の駒"""
        return Piece(piece_id, x, y, 2, 1, PieceType.GRANDMOTHER_COLOR, "祖母")
        
    @staticmethod
    def create_assistant(piece_id, x, y):
        """手代（1×1）- 小さな駒"""
        return Piece(piece_id, x, y, 1, 1, PieceType.ASSISTANT_COLOR, "手代")
        
    @staticmethod
    def create_dog(piece_id, x, y):
        """番犬（1×1）- 小さな駒"""
        return Piece(piece_id, x, y, 1, 1, PieceType.DOG_COLOR, "犬")
        
    @staticmethod
    def create_sister_in_law(piece_id, x, y):
        """兄嫁（1×1）- 小さな駒"""
        return Piece(piece_id, x, y, 1, 1, PieceType.SISTER_IN_LAW_COLOR, "嫁")
        
    @staticmethod
    def create_apprentice(piece_id, x, y, number=1):
        """丁稚（1×1）- 小さな駒"""
        return Piece(piece_id, x, y, 1, 1, PieceType.APPRENTICE_COLOR, f"丁{number}")