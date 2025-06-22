import pygame
import sys
from board import Board
from piece import Piece, PieceType

pygame.init()

WINDOW_WIDTH = 800
WINDOW_HEIGHT = 600
FPS = 60

WHITE = (255, 255, 255)
BLACK = (0, 0, 0)
GRAY = (128, 128, 128)
LIGHT_GRAY = (200, 200, 200)

class HakoiriMusumeGame:
    def __init__(self):
        self.screen = pygame.display.set_mode((WINDOW_WIDTH, WINDOW_HEIGHT))
        pygame.display.set_caption("箱入り娘の大家族")
        self.clock = pygame.time.Clock()
        self.running = True
        self.board = Board(6, 5)  # 6x5の盤面
        self.pieces = []
        self.selected_piece = None
        self.dragging = False
        self.drag_offset_x = 0
        self.drag_offset_y = 0
        self.possible_moves = []  # 選択された駒の移動可能位置
        self.hover_move = None    # ドラッグ中にハイライトされている移動先
        self.game_completed = False  # ゲーム完了フラグ
        self.move_count = 0       # 手数カウント
        self.setup_initial_pieces()
        
    def setup_initial_pieces(self):
        """
        初期の駒配置をセットアップ（合計15コマ）
        配置パターン：
        空父娘娘母空
        空父娘娘母空  
        手大大大大嫁
        丁番頭女中丁
        犬祖父祖母丁
        """
        piece_id = 1
        
        # 娘（2×2）- 中央（脱出目標駒）
        daughter = PieceType.create_daughter(piece_id, 2, 0)
        self.pieces.append(daughter)
        piece_id += 1
        
        # 父（1×2）- 左
        father = PieceType.create_father(piece_id, 1, 0)
        self.pieces.append(father)
        piece_id += 1
        
        # 母（1×2）- 右
        mother = PieceType.create_mother(piece_id, 4, 0)
        self.pieces.append(mother)
        piece_id += 1
        
        # 手代（1×1）- 左下
        assistant = PieceType.create_assistant(piece_id, 0, 2)
        self.pieces.append(assistant)
        piece_id += 1
        
        # 大番頭（4×1）- 横
        head_clerk = PieceType.create_head_clerk(piece_id, 1, 2)
        self.pieces.append(head_clerk)
        piece_id += 1
        
        # 兄嫁（1×1）- 右下
        sister_in_law = PieceType.create_sister_in_law(piece_id, 5, 2)
        self.pieces.append(sister_in_law)
        piece_id += 1
        
        # 丁稚1（1×1）- 左
        apprentice1 = PieceType.create_apprentice(piece_id, 0, 3, 1)
        self.pieces.append(apprentice1)
        piece_id += 1
        
        # 番頭（2×1）- 横向き
        clerk = PieceType.create_clerk(piece_id, 1, 3)
        self.pieces.append(clerk)
        piece_id += 1
        
        # 女中（2×1）- 横向き
        maid = PieceType.create_maid(piece_id, 3, 3)
        self.pieces.append(maid)
        piece_id += 1
        
        # 丁稚2（1×1）- 右
        apprentice2 = PieceType.create_apprentice(piece_id, 5, 3, 2)
        self.pieces.append(apprentice2)
        piece_id += 1
        
        # 番犬（1×1）- 左下
        dog = PieceType.create_dog(piece_id, 0, 4)
        self.pieces.append(dog)
        piece_id += 1
        
        # 祖父（2×1）- 横向き
        grandfather = PieceType.create_grandfather(piece_id, 1, 4)
        self.pieces.append(grandfather)
        piece_id += 1
        
        # 祖母（2×1）- 横向き
        grandmother = PieceType.create_grandmother(piece_id, 3, 4)
        self.pieces.append(grandmother)
        piece_id += 1
        
        # 丁稚3（1×1）- 右下
        apprentice3 = PieceType.create_apprentice(piece_id, 5, 4, 3)
        self.pieces.append(apprentice3)
        piece_id += 1
            
        # 盤面に駒を配置
        for piece in self.pieces:
            for cell_x, cell_y in piece.get_occupied_cells():
                self.board.grid[cell_y][cell_x] = piece.id
        
    def handle_events(self):
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                self.running = False
            elif event.type == pygame.MOUSEBUTTONDOWN:
                if event.button == 1:  # 左クリック
                    self.handle_mouse_down(event.pos)
            elif event.type == pygame.MOUSEBUTTONUP:
                if event.button == 1:  # 左クリック
                    self.handle_mouse_up(event.pos)
            elif event.type == pygame.MOUSEMOTION:
                if self.dragging:
                    self.handle_mouse_drag(event.pos)
    
    def get_piece_at_position(self, pos):
        """指定された画面座標にある駒を取得"""
        mouse_x, mouse_y = pos
        grid_x, grid_y = self.board.pixel_to_grid(mouse_x, mouse_y)
        
        if grid_x is None or grid_y is None:
            return None
            
        # 駒を逆順でチェック（後から描画された駒を優先）
        for piece in reversed(self.pieces):
            if (grid_x, grid_y) in piece.get_occupied_cells():
                return piece
        return None
    
    def handle_mouse_down(self, pos):
        """マウスダウン時の処理"""
        if self.game_completed:
            return
            
        piece = self.get_piece_at_position(pos)
        if piece:
            # 前の選択を解除
            if self.selected_piece:
                self.selected_piece.selected = False
            
            # 可能な移動先を取得
            possible_moves = piece.get_possible_moves(self.board)
            
            # 可能な移動先を保存
            self.possible_moves = possible_moves
            
            # 移動先が1つだけの場合は自動で移動
            if len(possible_moves) == 1:
                new_x, new_y = possible_moves[0]
                if piece.move_to(new_x, new_y, self.board):
                    self.move_count += 1
                    print(f"{piece.name}を自動で ({new_x}, {new_y}) に移動しました (手数: {self.move_count})")
                    self.check_win_condition()
                self.selected_piece = None
                self.possible_moves = []
                return
            
            # 移動先が0個または2個以上の場合は通常のドラッグモードに
            self.selected_piece = piece
            piece.selected = True
            self.dragging = True
            
            # ドラッグオフセットを計算
            piece_pixel_x, piece_pixel_y = self.board.grid_to_pixel(piece.x, piece.y)
            self.drag_offset_x = pos[0] - piece_pixel_x
            self.drag_offset_y = pos[1] - piece_pixel_y
            
            # デバッグ情報を出力
            if len(possible_moves) == 0:
                print(f"{piece.name}は移動できません")
            else:
                print(f"{piece.name}の可能な移動先: {possible_moves}")
    
    def handle_mouse_drag(self, pos):
        """マウスドラッグ時の処理"""
        if self.selected_piece and self.dragging:
            # ドロップ位置を計算
            drop_pixel_x = pos[0] - self.drag_offset_x
            drop_pixel_y = pos[1] - self.drag_offset_y
            
            # ピクセル座標をグリッド座標に変換
            drop_grid_x = round((drop_pixel_x - self.board.start_x) / self.board.cell_size)
            drop_grid_y = round((drop_pixel_y - self.board.start_y) / self.board.cell_size)
            
            # ドラッグ位置が可能な移動先の中にあるかチェック
            self.hover_move = None
            for move_x, move_y in self.possible_moves:
                if move_x == drop_grid_x and move_y == drop_grid_y:
                    self.hover_move = (move_x, move_y)
                    break
    
    def handle_mouse_up(self, pos):
        """マウスアップ時の処理"""
        if self.dragging and self.selected_piece:
            # ハイライトされた移動先がある場合はそこに移動
            if self.hover_move:
                move_x, move_y = self.hover_move
                if self.selected_piece.move_to(move_x, move_y, self.board):
                    self.move_count += 1
                    print(f"{self.selected_piece.name}を ({move_x}, {move_y}) に移動しました (手数: {self.move_count})")
                    self.check_win_condition()
                else:
                    print("移動に失敗しました")
            else:
                print("有効な移動先にドロップしてください")
                
        # ドラッグ状態をリセット
        self.dragging = False
        if self.selected_piece:
            self.selected_piece.selected = False
            self.selected_piece = None
        self.possible_moves = []
        self.hover_move = None
    
    def check_win_condition(self):
        """勝利条件をチェック"""
        # 娘の駒を探す
        daughter = None
        for piece in self.pieces:
            if piece.name == "娘":
                daughter = piece
                break
        
        if daughter:
            # 娘が脱出位置（2,3）に到達したかチェック
            exit_x, exit_y = 2, 3
            if daughter.x == exit_x and daughter.y == exit_y:
                self.game_completed = True
                print(f"🎉 ゲームクリア！娘を脱出させました！")
                print(f"総手数: {self.move_count}手")
                pygame.display.set_caption(f"箱入り娘の大家族 - ゲームクリア！({self.move_count}手)")
                
    def update(self):
        pass
        
    def draw(self):
        self.screen.fill(WHITE)
        self.board.draw(self.screen)
        
        # 移動可能マスのオーバーレイを描画
        self.draw_move_overlays()
        
        # 駒を描画
        for piece in self.pieces:
            piece.draw(self.screen, self.board)
            
        pygame.display.flip()
    
    def draw_move_overlays(self):
        """移動可能マスのオーバーレイを描画"""
        if not self.possible_moves:
            return
            
        # 半透明の青色でオーバーレイ
        overlay_color = (100, 150, 255, 80)  # 青色、透明度80
        highlight_color = (255, 200, 0, 150)  # 黄色、透明度150
        
        for move_x, move_y in self.possible_moves:
            # 駒のサイズ分のオーバーレイを描画
            if self.selected_piece:
                for dy in range(self.selected_piece.height):
                    for dx in range(self.selected_piece.width):
                        cell_x = move_x + dx
                        cell_y = move_y + dy
                        
                        if self.board.is_valid_position(cell_x, cell_y):
                            pixel_x, pixel_y = self.board.grid_to_pixel(cell_x, cell_y)
                            
                            # ハイライト色か通常のオーバーレイ色かを決定
                            if self.hover_move and self.hover_move == (move_x, move_y):
                                color = highlight_color
                            else:
                                color = overlay_color
                            
                            # 半透明の矩形を描画
                            overlay_surface = pygame.Surface((self.board.cell_size, self.board.cell_size))
                            overlay_surface.set_alpha(color[3])
                            overlay_surface.fill(color[:3])
                            self.screen.blit(overlay_surface, (pixel_x, pixel_y))
        
    def run(self):
        while self.running:
            self.handle_events()
            self.update()
            self.draw()
            self.clock.tick(FPS)
        
        pygame.quit()
        sys.exit()

if __name__ == "__main__":
    game = HakoiriMusumeGame()
    game.run()