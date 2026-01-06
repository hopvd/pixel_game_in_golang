# RPG in Go
A [video series](https://youtube.com/playlist?list=PLvN4CrYN-8i7xnODFyCMty6ossz4eW0Cn&si=xuvPY13Kodf5nPzH) by me!

## Game Features

- **Player Movement**: Use arrow keys to move your ninja character
- **Combat System**: Press Space to throw shurikens at enemies
- **Enemy AI**: Enemies chase the player when within range
- **Health System**: 
  - Player has 3 health points
  - Enemies have 3 health points
  - Health bars displayed above characters
- **Damage System**: 
  - Player loses health when colliding with enemies
  - Enemies lose health when hit by shurikens
  - Dead enemies stop moving and only show their head
- **Items**: Collect potions to restore health
- **Game Over**: Game ends when player health reaches 0
- **Restart**: Press R to restart after game over

## How to Run 2222222

### Prerequisites
- Go 1.22.4 or later

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd rpg-in-golang
```

2. Install dependencies:
```bash
go mod download
```

3. Run the game:
```bash
go run .
```

## Controls

- **Arrow Keys**: Move player (Up, Down, Left, Right)
- **Space**: Throw shuriken
- **R**: Restart game (when game over)
- **ESC**: Exit game

## Repository Structure

This project uses git branches to manage the different episodes. Click on the branch labelled `main` in the topleft and select the appropriate episode:
![Screenshot of selecting a branch](./github_assets/branches.png)

# License

All code is licensed under [MIT](./LICENSE)

# Support

All content revolving around this series is given for free. It would be a huge help to support in ways that you can, including:

- Subscribing to my [channel](https://youtube.com/@codingwithsphere)
- Sharing my content
- Supporting me on [Patreon](https://patreon.com/codingwithsphere)