package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// the base struct for all our moving, drawn entities
type Sprite struct {
	Img  *ebiten.Image
	X, Y float64
}

type Player struct {
	*Sprite
	Health    uint
	MaxHealth uint
	// Cooldown to prevent continuous damage
	damageCooldown int
}

type Enemy struct {
	*Sprite
	FollowsPlayer bool
	Health        uint
	MaxHealth     uint
	Scale         float64 // Scale factor for larger enemies
}

type Potion struct {
	*Sprite
	AmtHeal uint
}

type Shuriken struct {
	X, Y       float64
	VelX, VelY float64 // Velocity
	Distance   float64 // Distance traveled
	MaxRange   float64 // Maximum range
}

type Game struct {
	// the image and position variables for our player
	player      *Player
	enemies     []*Enemy
	potions     []*Potion
	shurikens   []*Shuriken
	tilemapJSON *TilemapJSON
	tilemapImg  *ebiten.Image
	gameOver    bool
	// Frame counter for cooldown
	frameCount int
	// Track previous key state to detect key press
	spacePressed bool
	// Level system
	currentLevel int
	// Initial state for reset
	initialPlayerX, initialPlayerY float64
	initialPlayerHealth            uint
	initialEnemyPositions          []struct{ X, Y float64 }
	initialEnemyHealth             uint
	initialPotionData              []struct {
		X, Y    float64
		AmtHeal uint
	}
	// Store images for reset
	playerImg   *ebiten.Image
	skeletonImg *ebiten.Image
	potionImg   *ebiten.Image
	shurikenImg *ebiten.Image
}

func (g *Game) Update() error {
	// Increment frame counter
	g.frameCount++

	// If game is over, check for restart key
	if g.gameOver {
		// Check if R key is pressed to restart
		if ebiten.IsKeyPressed(ebiten.KeyR) {
			g.resetGame()
		}
		return nil
	}

	// Decrease damage cooldown
	if g.player.damageCooldown > 0 {
		g.player.damageCooldown--
	}

	// move the player based on keyboar input (left, right, up down)
	movedX, movedY := 0.0, 0.0
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.player.X -= 2
		movedX = -2
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.player.X += 2
		movedX = 2
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.player.Y -= 2
		movedY = -2
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.player.Y += 2
		movedY = 2
	}

	// Handle shuriken shooting with Space key
	currentSpacePressed := ebiten.IsKeyPressed(ebiten.KeySpace)
	if currentSpacePressed && !g.spacePressed {
		// Space key just pressed, create a new shuriken
		// Determine direction based on last movement, or default to right
		velX, velY := 3.0, 0.0 // Default to right
		if movedX != 0 || movedY != 0 {
			// Normalize direction
			length := math.Sqrt(movedX*movedX + movedY*movedY)
			velX = (movedX / length) * 3.0
			velY = (movedY / length) * 3.0
		}

		shuriken := &Shuriken{
			X:        g.player.X + 8, // Center of player
			Y:        g.player.Y + 8, // Center of player
			VelX:     velX,
			VelY:     velY,
			Distance: 0,
			MaxRange: 100.0, // Short range
		}
		g.shurikens = append(g.shurikens, shuriken)
	}
	g.spacePressed = currentSpacePressed

	// Update shurikens and check collision with enemies
	for i := len(g.shurikens) - 1; i >= 0; i-- {
		shuriken := g.shurikens[i]
		shuriken.X += shuriken.VelX
		shuriken.Y += shuriken.VelY
		shuriken.Distance += math.Sqrt(shuriken.VelX*shuriken.VelX + shuriken.VelY*shuriken.VelY)

		// Check collision with enemies
		hitEnemy := false
		for _, enemy := range g.enemies {
			if enemy.Health > 0 {
				// Check collision between shuriken and enemy
				if checkShurikenEnemyCollision(shuriken, enemy.Sprite, enemy.Scale) {
					// Enemy takes damage
					if enemy.Health > 0 {
						enemy.Health--
						fmt.Printf("Enemy hit! Health: %d/%d\n", enemy.Health, enemy.MaxHealth)
					}
					hitEnemy = true
					break
				}
			}
		}

		// Remove shuriken if it hits an enemy or exceeds max range
		if hitEnemy || shuriken.Distance >= shuriken.MaxRange {
			g.shurikens = append(g.shurikens[:i], g.shurikens[i+1:]...)
		}
	}

	// add behavior to the enemies
	for _, enemy := range g.enemies {
		// Only move and interact if enemy is alive
		if enemy.Health > 0 {
			// 1. Calculate distance between Ninja and Skeleton (Pythagoras)
			dx := g.player.X - enemy.X
			dy := g.player.Y - enemy.Y
			distance := math.Sqrt(dx*dx + dy*dy)

			// 2. Only chase if distance is less than 50 pixels
			if distance < 50 {
				if enemy.X < g.player.X {
					enemy.X += 1
				} else if enemy.X > g.player.X {
					enemy.X -= 1
				}
				if enemy.Y < g.player.Y {
					enemy.Y += 1
				} else if enemy.Y > g.player.Y {
					enemy.Y -= 1
				}
			}

			// Check collision between player and enemy with smaller collision area
			if checkPlayerEnemyCollision(g.player.Sprite, enemy.Sprite, enemy.Scale) {
				// Only damage if cooldown is 0
				if g.player.damageCooldown <= 0 {
					if g.player.Health > 0 {
						g.player.Health--
						fmt.Printf("Player took damage! Health: %d/%d\n", g.player.Health, g.player.MaxHealth)
						// Set cooldown to 60 frames (1 second at 60 FPS)
						g.player.damageCooldown = 60
					}
					// Check if player is dead
					if g.player.Health == 0 {
						g.gameOver = true
						fmt.Println("Game Over! You lost!")
					}
				}
			}
		}
	}

	// handle simple potion functionality
	for i := 0; i < len(g.potions); i++ {
		potion := g.potions[i]

		if checkCollision(g.player.Sprite, potion.Sprite) {
			// Heal player
			g.player.Health += potion.AmtHeal
			fmt.Printf("Picked up potion! Health: %d\n", g.player.Health)

			// Remove collected potion from the list
			g.potions = append(g.potions[:i], g.potions[i+1:]...)
			i-- // Decrease index i to not skip the next element
		}
	}

	// Check if all enemies are defeated
	if g.checkAllEnemiesDefeated() {
		g.loadNextLevel()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

	// fill the screen with a nice sky color
	screen.Fill(color.RGBA{120, 180, 255, 255})

	opts := ebiten.DrawImageOptions{}

	// loop over the layers
	for _, layer := range g.tilemapJSON.Layers {
		// loop over the tiles in the layer data
		for index, id := range layer.Data {

			// get the tile position of the tile
			x := index % layer.Width
			y := index / layer.Width

			// convert the tile position to pixel position
			x *= 16
			y *= 16

			// get the position on the image where the tile id is
			srcX := (id - 1) % 22
			srcY := (id - 1) / 22

			// convert the src tile pos to pixel src position
			srcX *= 16
			srcY *= 16

			// set the drawimageoptions to draw the tile at x, y
			opts.GeoM.Translate(float64(x), float64(y))

			// draw the tile
			screen.DrawImage(
				// cropping out the tile that we want from the spritesheet
				g.tilemapImg.SubImage(image.Rect(srcX, srcY, srcX+16, srcY+16)).(*ebiten.Image),
				&opts,
			)

			// reset the opts for the next tile
			opts.GeoM.Reset()
		}
	}

	// set the translation of our drawImageOptions to the player's position
	opts.GeoM.Translate(g.player.X, g.player.Y)

	// draw the player
	screen.DrawImage(
		// grab a subimage of the spritesheet
		g.player.Img.SubImage(
			image.Rect(0, 0, 16, 16),
		).(*ebiten.Image),
		&opts,
	)

	opts.GeoM.Reset()

	for _, enemy := range g.enemies {
		opts.GeoM.Reset()

		// Apply scale first, then translate
		if enemy.Scale != 1.0 {
			opts.GeoM.Scale(enemy.Scale, enemy.Scale)
		}
		opts.GeoM.Translate(enemy.X, enemy.Y)

		if enemy.Health > 0 {
			// Draw full enemy sprite when alive
			screen.DrawImage(
				enemy.Img.SubImage(
					image.Rect(0, 0, 16, 16),
				).(*ebiten.Image),
				&opts,
			)
		} else {
			// Draw only the head (top 8x8 pixels) when dead
			opts.GeoM.Reset()
			if enemy.Scale != 1.0 {
				opts.GeoM.Scale(enemy.Scale, enemy.Scale)
			}
			opts.GeoM.Translate(enemy.X, enemy.Y+4*enemy.Scale) // Move down a bit to center the head
			screen.DrawImage(
				enemy.Img.SubImage(
					image.Rect(0, 0, 16, 8), // Only top half (head)
				).(*ebiten.Image),
				&opts,
			)
		}

		opts.GeoM.Reset()
	}

	opts.GeoM.Reset()

	// Draw shurikens
	for _, shuriken := range g.shurikens {
		opts.GeoM.Reset()
		// Center the shuriken image (assuming 8x8 size)
		opts.GeoM.Translate(shuriken.X-4, shuriken.Y-4)
		screen.DrawImage(g.shurikenImg, &opts)
	}

	opts.GeoM.Reset()

	for _, sprite := range g.potions {
		opts.GeoM.Translate(sprite.X, sprite.Y)

		screen.DrawImage(
			sprite.Img.SubImage(
				image.Rect(0, 0, 16, 16),
			).(*ebiten.Image),
			&opts,
		)

		opts.GeoM.Reset()
	}

	// Draw health bars
	drawHealthBar(screen, g.player.X, g.player.Y-6, g.player.Health, g.player.MaxHealth, color.RGBA{0, 255, 0, 255}) // Green for player

	for _, enemy := range g.enemies {
		// Only draw health bar for alive enemies
		if enemy.Health > 0 {
			// Adjust health bar position based on enemy scale
			healthBarY := enemy.Y - 6*enemy.Scale
			drawHealthBar(screen, enemy.X, healthBarY, enemy.Health, enemy.MaxHealth, color.RGBA{255, 0, 0, 255}) // Red for enemies
		}
	}

	// Display level info
	levelText := fmt.Sprintf("Level: %d", g.currentLevel+1)
	ebitenutil.DebugPrintAt(screen, levelText, 10, 10)

	// Display Game Over message if player lost
	if g.gameOver {
		ebitenutil.DebugPrint(screen, "GAME OVER!\nYou lost!\nPress R to restart\nPress ESC to exit")
	}

}

func checkCollision(s1, s2 *Sprite) bool {
	// Assume each object (player, potion) has a size of 16x16 pixels
	return s1.X < s2.X+16 &&
		s1.X+16 > s2.X &&
		s1.Y < s2.Y+16 &&
		s1.Y+16 > s2.Y
}

// checkPlayerEnemyCollision checks collision with a smaller area for more precise collision
func checkPlayerEnemyCollision(player, enemy *Sprite, enemyScale float64) bool {
	// Use smaller collision area (8x8 pixels) - player and enemy must be closer to collide
	collisionSize := 8.0
	enemySize := 16.0 * enemyScale
	// Center the collision box within the sprite
	playerOffset := (16.0 - collisionSize) / 2.0
	enemyOffset := (enemySize - collisionSize) / 2.0

	playerCenterX := player.X + playerOffset
	playerCenterY := player.Y + playerOffset
	enemyCenterX := enemy.X + enemyOffset
	enemyCenterY := enemy.Y + enemyOffset

	return playerCenterX < enemyCenterX+collisionSize &&
		playerCenterX+collisionSize > enemyCenterX &&
		playerCenterY < enemyCenterY+collisionSize &&
		playerCenterY+collisionSize > enemyCenterY
}

// checkShurikenEnemyCollision checks collision between shuriken and enemy
func checkShurikenEnemyCollision(shuriken *Shuriken, enemy *Sprite, enemyScale float64) bool {
	// Shuriken is 8x8, enemy size depends on scale
	shurikenSize := 8.0
	enemySize := 16.0 * enemyScale
	return shuriken.X < enemy.X+enemySize &&
		shuriken.X+shurikenSize > enemy.X &&
		shuriken.Y < enemy.Y+enemySize &&
		shuriken.Y+shurikenSize > enemy.Y
}

// checkAllEnemiesDefeated checks if all enemies are dead
func (g *Game) checkAllEnemiesDefeated() bool {
	for _, enemy := range g.enemies {
		if enemy.Health > 0 {
			return false
		}
	}
	return len(g.enemies) > 0 // Only return true if there were enemies to begin with
}

// loadNextLevel loads the next level
func (g *Game) loadNextLevel() {
	g.currentLevel++
	fmt.Printf("Level %d completed! Loading level %d...\n", g.currentLevel-1, g.currentLevel)

	// Clear all shurikens
	g.shurikens = []*Shuriken{}

	// Reset player position to center
	g.player.X = 160.0
	g.player.Y = 120.0

	// Load enemies based on level
	if g.currentLevel == 1 {
		// Level 1: 2 enemies with 10 health
		g.enemies = []*Enemy{
			{
				&Sprite{
					Img: g.skeletonImg,
					X:   100.0,
					Y:   100.0,
				},
				true,
				10,  // Health
				10,  // MaxHealth
				1.0, // Scale (normal size)
			},
			{
				&Sprite{
					Img: g.skeletonImg,
					X:   150.0,
					Y:   50.0,
				},
				true,
				10,  // Health
				10,  // MaxHealth
				1.0, // Scale (normal size)
			},
		}
	} else if g.currentLevel == 2 {
		// Level 2: 1 large enemy with 50 health
		g.enemies = []*Enemy{
			{
				&Sprite{
					Img: g.skeletonImg,
					X:   160.0,
					Y:   120.0,
				},
				true,
				50,  // Health
				50,  // MaxHealth
				2.0, // Scale (2x size - larger enemy)
			},
		}
		fmt.Println("Boss enemy appeared!")
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 240
}

// drawHealthBar draws a health bar above a sprite
func drawHealthBar(screen *ebiten.Image, x, y float64, currentHealth, maxHealth uint, barColor color.RGBA) {
	if maxHealth == 0 {
		return
	}

	barWidth := 16.0
	barHeight := 2.0
	borderWidth := 1.0

	// Draw border (black background)
	borderImg := ebiten.NewImage(int(barWidth+2*borderWidth), int(barHeight+2*borderWidth))
	borderImg.Fill(color.RGBA{0, 0, 0, 255})

	opts := ebiten.DrawImageOptions{}
	opts.GeoM.Translate(x-borderWidth, y-borderWidth)
	screen.DrawImage(borderImg, &opts)

	// Draw health bar
	if currentHealth > 0 {
		healthPercent := float64(currentHealth) / float64(maxHealth)
		healthWidth := barWidth * healthPercent

		healthImg := ebiten.NewImage(int(healthWidth), int(barHeight))
		healthImg.Fill(barColor)

		opts.GeoM.Reset()
		opts.GeoM.Translate(x, y)
		screen.DrawImage(healthImg, &opts)
	}
}

// resetGame resets the game to its initial state
func (g *Game) resetGame() {
	// Reset level
	g.currentLevel = 0

	// Reset player position and health
	g.player.X = g.initialPlayerX
	g.player.Y = g.initialPlayerY
	g.player.Health = g.initialPlayerHealth
	g.player.damageCooldown = 0
	g.frameCount = 0

	// Reset enemies to initial positions and health (level 1)
	g.enemies = []*Enemy{
		{
			&Sprite{
				Img: g.skeletonImg,
				X:   100.0,
				Y:   100.0,
			},
			true,
			10,  // Health
			10,  // MaxHealth
			1.0, // Scale (normal size)
		},
		{
			&Sprite{
				Img: g.skeletonImg,
				X:   150.0,
				Y:   50.0,
			},
			true,
			10,  // Health
			10,  // MaxHealth
			1.0, // Scale (normal size)
		},
	}

	// Reset potions - recreate from initial state
	g.potions = make([]*Potion, len(g.initialPotionData))
	for i, data := range g.initialPotionData {
		g.potions[i] = &Potion{
			Sprite: &Sprite{
				Img: g.potionImg,
				X:   data.X,
				Y:   data.Y,
			},
			AmtHeal: data.AmtHeal,
		}
	}

	// Reset shurikens
	g.shurikens = []*Shuriken{}
	g.spacePressed = false

	// Reset game over state
	g.gameOver = false
	fmt.Println("Game restarted!")
}

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Hello, World!")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// load the image from file
	playerImg, _, err := ebitenutil.NewImageFromFile("assets/images/ninja.png")
	if err != nil {
		// handle error
		log.Fatal(err)
	}
	// load the image from file
	skeletonImg, _, err := ebitenutil.NewImageFromFile("assets/images/skeleton.png")
	if err != nil {
		// handle error
		log.Fatal(err)
	}

	potionImg, _, err := ebitenutil.NewImageFromFile("assets/images/potion.png")
	if err != nil {
		// handle error
		log.Fatal(err)
	}

	tilemapImg, _, err := ebitenutil.NewImageFromFile("assets/images/TilesetFloor.png")
	if err != nil {
		// handle error
		log.Fatal(err)
	}

	tilemapJSON, err := NewTilemapJSON("assets/maps/spawn.json")
	if err != nil {
		log.Fatal(err)
	}

	// Create shuriken image (8x8 pixels)
	shurikenImg := ebiten.NewImage(8, 8)
	// Draw a simple shuriken shape (star-like with 4 blades)
	// Fill background with transparent (or dark)
	shurikenImg.Fill(color.RGBA{0, 0, 0, 0})

	// Draw shuriken blades (4-pointed star)
	// Center point
	shurikenImg.Set(4, 4, color.RGBA{200, 200, 200, 255})

	// Top blade
	shurikenImg.Set(4, 0, color.RGBA{255, 255, 255, 255})
	shurikenImg.Set(4, 1, color.RGBA{220, 220, 220, 255})
	shurikenImg.Set(4, 2, color.RGBA{200, 200, 200, 255})
	shurikenImg.Set(4, 3, color.RGBA{180, 180, 180, 255})

	// Bottom blade
	shurikenImg.Set(4, 5, color.RGBA{180, 180, 180, 255})
	shurikenImg.Set(4, 6, color.RGBA{200, 200, 200, 255})
	shurikenImg.Set(4, 7, color.RGBA{220, 220, 220, 255})

	// Left blade
	shurikenImg.Set(0, 4, color.RGBA{255, 255, 255, 255})
	shurikenImg.Set(1, 4, color.RGBA{220, 220, 220, 255})
	shurikenImg.Set(2, 4, color.RGBA{200, 200, 200, 255})
	shurikenImg.Set(3, 4, color.RGBA{180, 180, 180, 255})

	// Right blade
	shurikenImg.Set(5, 4, color.RGBA{180, 180, 180, 255})
	shurikenImg.Set(6, 4, color.RGBA{200, 200, 200, 255})
	shurikenImg.Set(7, 4, color.RGBA{220, 220, 220, 255})

	// Diagonal accents
	shurikenImg.Set(1, 1, color.RGBA{150, 150, 150, 255})
	shurikenImg.Set(6, 6, color.RGBA{150, 150, 150, 255})
	shurikenImg.Set(1, 6, color.RGBA{150, 150, 150, 255})
	shurikenImg.Set(6, 1, color.RGBA{150, 150, 150, 255})

	// Initial positions and states
	initialPlayerX := 50.0
	initialPlayerY := 50.0
	initialPlayerHealth := uint(3)

	initialEnemyPositions := []struct{ X, Y float64 }{
		{X: 100.0, Y: 100.0},
		{X: 150.0, Y: 50.0},
	}
	initialEnemyHealth := uint(10)

	initialPotionData := []struct {
		X, Y    float64
		AmtHeal uint
	}{
		{X: 210.0, Y: 100.0, AmtHeal: 1},
	}

	game := Game{
		player: &Player{
			Sprite: &Sprite{
				Img: playerImg,
				X:   initialPlayerX,
				Y:   initialPlayerY,
			},
			Health:    initialPlayerHealth,
			MaxHealth: initialPlayerHealth,
		},
		enemies: []*Enemy{
			{
				&Sprite{
					Img: skeletonImg,
					X:   100.0,
					Y:   100.0,
				},
				true,
				10,  // Health
				10,  // MaxHealth
				1.0, // Scale (normal size)
			},
			{
				&Sprite{
					Img: skeletonImg,
					X:   150.0,
					Y:   50.0,
				},
				true,
				10,  // Health
				10,  // MaxHealth
				1.0, // Scale (normal size)
			},
		},
		currentLevel: 0, // Start at level 0 (will be level 1 when displayed)
		potions: []*Potion{
			{
				&Sprite{
					Img: potionImg,
					X:   210.0,
					Y:   100.0,
				},
				1.0,
			},
		},
		tilemapJSON:           tilemapJSON,
		tilemapImg:            tilemapImg,
		initialPlayerX:        initialPlayerX,
		initialPlayerY:        initialPlayerY,
		initialPlayerHealth:   initialPlayerHealth,
		initialEnemyPositions: initialEnemyPositions,
		initialEnemyHealth:    initialEnemyHealth,
		initialPotionData:     initialPotionData,
		playerImg:             playerImg,
		skeletonImg:           skeletonImg,
		potionImg:             potionImg,
		shurikenImg:           shurikenImg,
	}

	if err := ebiten.RunGame(&game); err != nil {
		log.Fatal(err)
	}
}
