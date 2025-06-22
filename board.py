import pygame

class Board:
    def __init__(self, width=7, height=9):
        self.width = width  # 盤面の幅（マス数）
        self.height = height  # 盤面の高さ（マス数）
        self.cell_size = 60  # 各マスのサイズ（ピクセル）
        self.margin = 50  # 盤面の余白
        
        # 盤面の実際の描画サイズ
        self.board_width = self.width * self.cell_size
        self.board_height = self.height * self.cell_size
        
        # 盤面の左上角の座標
        self.start_x = self.margin
        self.start_y = self.margin
        
        # 盤面の状態（0は空、その他は駒のID）
        self.grid = [[0 for _ in range(self.width)] for _ in range(self.height)]
        
    def get_cell_rect(self, x, y):
        """指定されたグリッド座標のセルの矩形を返す"""
        rect_x = self.start_x + x * self.cell_size
        rect_y = self.start_y + y * self.cell_size
        return pygame.Rect(rect_x, rect_y, self.cell_size, self.cell_size)
        
    def grid_to_pixel(self, grid_x, grid_y):
        """グリッド座標をピクセル座標に変換"""
        pixel_x = self.start_x + grid_x * self.cell_size
        pixel_y = self.start_y + grid_y * self.cell_size
        return pixel_x, pixel_y
        
    def pixel_to_grid(self, pixel_x, pixel_y):
        """ピクセル座標をグリッド座標に変換"""
        if (pixel_x < self.start_x or pixel_x >= self.start_x + self.board_width or
            pixel_y < self.start_y or pixel_y >= self.start_y + self.board_height):
            return None, None
            
        grid_x = (pixel_x - self.start_x) // self.cell_size
        grid_y = (pixel_y - self.start_y) // self.cell_size
        return grid_x, grid_y
        
    def is_valid_position(self, x, y):
        """指定された座標が盤面内かどうかチェック"""
        return 0 <= x < self.width and 0 <= y < self.height
        
    def is_empty(self, x, y):
        """指定された座標が空かどうかチェック"""
        if not self.is_valid_position(x, y):
            return False
        return self.grid[y][x] == 0
        
    def draw(self, screen):
        """盤面を描画"""
        # 背景色
        WHITE = (255, 255, 255)
        BLACK = (0, 0, 0)
        LIGHT_GRAY = (200, 200, 200)
        
        # 盤面の背景
        board_rect = pygame.Rect(self.start_x, self.start_y, 
                                self.board_width, self.board_height)
        pygame.draw.rect(screen, LIGHT_GRAY, board_rect)
        pygame.draw.rect(screen, BLACK, board_rect, 2)
        
        # グリッド線を描画
        for x in range(self.width + 1):
            start_pos = (self.start_x + x * self.cell_size, self.start_y)
            end_pos = (self.start_x + x * self.cell_size, 
                      self.start_y + self.board_height)
            pygame.draw.line(screen, BLACK, start_pos, end_pos, 1)
            
        for y in range(self.height + 1):
            start_pos = (self.start_x, self.start_y + y * self.cell_size)
            end_pos = (self.start_x + self.board_width, 
                      self.start_y + y * self.cell_size)
            pygame.draw.line(screen, BLACK, start_pos, end_pos, 1)
            
        # 脱出口を描画
        self.draw_exit_area(screen)
    
    def draw_exit_area(self, screen):
        """脱出口を描画"""
        exit_x, exit_y = 2, 3
        exit_width, exit_height = 2, 2
        
        pixel_x, pixel_y = self.grid_to_pixel(exit_x, exit_y)
        width_pixels = exit_width * self.cell_size
        height_pixels = exit_height * self.cell_size
        
        # 脱出エリアの背景（薄い緑色）
        exit_surface = pygame.Surface((width_pixels, height_pixels))
        exit_surface.set_alpha(80)
        exit_surface.fill((144, 238, 144))
        screen.blit(exit_surface, (pixel_x, pixel_y))
        
        # 脱出エリアの枠線（緑色、破線風）
        GREEN = (50, 205, 50)
        pygame.draw.rect(screen, GREEN, 
                        pygame.Rect(pixel_x, pixel_y, width_pixels, height_pixels), 3)
        
        # 「脱出口」のテキスト
        try:
            font = pygame.font.Font(None, 24)
            text = font.render("脱出口", True, (34, 139, 34))
            text_rect = text.get_rect(center=(pixel_x + width_pixels//2, 
                                            pixel_y + height_pixels//2))
            
            # テキストの背景
            bg_rect = text_rect.inflate(10, 6)
            bg_surface = pygame.Surface((bg_rect.width, bg_rect.height))
            bg_surface.set_alpha(200)
            bg_surface.fill((255, 255, 255))
            screen.blit(bg_surface, bg_rect)
            
            screen.blit(text, text_rect)
        except:
            # フォントが利用できない場合はスキップ
            pass