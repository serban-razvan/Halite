package main

import (
    "bufio"
    "container/heap"
    "fmt"
    "io"
    "log"
    "os"
    "math"
    "sort"
    "strings"
    "strconv"
    "time"
)

var Rx [][]int

var botname = "shummie v48-1-1-Go"

var Height, Width int

type Square struct {
    X, Y int
    Strength, Production, Owner float64
    Vertex int
    Target *Square
    Width, Height int
    Game *Game
    North, South, East, West *Square
    Neighbors []*Square
    MovingHere map[int]*Square
    Move int
    ResetStatus bool
}

type Direction int

const (
    NORTH Direction = iota
    EAST
    SOUTH
    WEST
    STILL
)


type Game struct {
    Width, Height int
    MyID, StartingPlayerCount float64
    MaxTurns, TurnsLeft float64
    PercentOwned float64
    Buildup float64
    MoveMap [][]float64
    ProductionMap, StrengthMap, OwnerMap [][]float64
    StrengthMap1, StrengthMap01, ProductionMap1, ProductionMap01 [][]float64
    IsOwnedMap, IsNeutralMap, IsEnemyMap [][]float64
    DistanceFromBorder, DistanceFromOwned, DistanceFromCombat [][]float64
    BorderMap, CombatZoneMap [][]float64
    DistanceMapNoDecay [][][][]float64
    DijkstraRecoveryCosts [][][][]float64
    DijkstraRecoveryPaths [][][][]int
    DijkstraRecoveryCosts2 [][][][]float64
    DijkstraRecoveryPaths2 [][][][]int
    DijkstraRecoveryCosts3 [][][][]float64
    DijkstraRecoveryPaths3 [][][][]int
    DijkstraRecoveryDone3 [][][][]bool

    RecoveryCostMap, GlobalContributionMap, ValueMap [][]float64
    Frame, Phase int
    Reader *bufio.Reader
    Writer io.Writer
    Squares [][]*Square
}

func NewGame() Game {
    game := Game{
        Reader: bufio.NewReader(os.Stdin),
        Writer: os.Stdout,
    }
    game.MyID = float64(game.getInt())
    game.deserializeMapSize()
    Height = game.Height
    Width = game.Width
    game.deserializeProductions()

    game.Squares = make([][]*Square, game.Width)
    for x := 0; x < game.Width; x++ {
        game.Squares[x] = make([]*Square, game.Height)
        for y := 0; y < game.Height; y++ {
            game.Squares[x][y] = &Square{X:x, Y:y, Production:game.ProductionMap[x][y], Vertex:x * game.Height + y, Width:game.Width, Height:game.Height, Game:&game}
        }
    }
    for x := 0; x < game.Width; x++ {
        for y := 0 ; y < game.Height; y++ {
            game.Squares[x][y].afterInitUpdate()
        }
    }
    game.Frame = -1

    game.OwnerMap = make([][]float64, game.Width)
    for x := 0; x < game.Width; x++ {
        game.OwnerMap[x] = make([]float64, game.Height)
    }
    game.StrengthMap = make([][]float64, game.Width)
    for x := 0; x < game.Width; x++ {
        game.StrengthMap[x] = make([]float64, game.Height)
    }
    game.MoveMap = make([][]float64, game.Width)
    for x := 0; x < game.Width; x++ {
        game.MoveMap[x] = make([]float64, game.Height)
    }
    game.getFrame()

    game.StartingPlayerCount = Max2d(game.OwnerMap)

    game.MaxTurns = 10 * math.Pow(float64(game.Width * game.Height), 0.5)

    game.createOneTimeMaps()
    game.setConfigs()

    fmt.Println(botname)

    return game
}

func (g *Game) getFrame() {
    // Updates the map information from the latest frame provided by the game environment
    mapString := g.getString()

    // The state of the map (including owner and strength values, but excluding production values) is sent in the following way:
    // One integer, COUNTER, representing the number of tiles with the same owner consecutively.
    // One integer, OWNER, representing the owner of the tiles COUNTER encodes.
    // The above repeats until the COUNTER total is equal to the area of the map.
    // It fills in the map from row 1 to row Height and within a row from column 1 to column Width.
    // Please be aware that the top row is the first row, as Halite uses screen-type coordinates.
    splitString := strings.Split(mapString, " ")


    var x, y, owner, counter int

    for y != g.Height {
        counter, splitString = int_str_array_pop(splitString)
        owner, splitString = int_str_array_pop(splitString)
        for a := 0; a < counter; a++ {
            g.OwnerMap[x][y] = float64(owner)
            x += 1
            if x == g.Width {
                x = 0
                y += 1
            }
        }
    }

    for y := 0; y < g.Height; y++ {
        for x := 0; x < g.Width; x++ {
            var strValue int
            strValue, splitString = int_str_array_pop(splitString)
            g.StrengthMap[x][y] = float64(strValue)
            g.MoveMap[x][y] = -1  // Reset the move map
            g.Squares[x][y].update(g.OwnerMap[x][y], g.StrengthMap[x][y])
        }
    }

    g.Frame += 1
}

func (g *Game) setConfigs() {
    g.Buildup = 5
    g.Phase = 0  // Temporary, we might get rid of this in this version
}

func (g *Game) createOneTimeMaps() {
    g.DistanceMapNoDecay = g.createDistanceMap(0)
    g.StrengthMap1 = MaxAcross2d(g.StrengthMap, 1)
    g.StrengthMap01 = MaxAcross2d(g.StrengthMap, 0.1)
    g.ProductionMap1 = MaxAcross2d(g.ProductionMap, 1)
    g.ProductionMap01 = MaxAcross2d(g.ProductionMap, 0.1)
    start := time.Now()
    g.createDijkstraMaps()

    end := time.Since(start)
    log.Println("Dijkstra took %s", end)
    start = time.Now()
    g.createDijkstraMaps2()
    end = time.Since(start)
    log.Println("Dijkstra2 took %s", end)
    start = time.Now()
    // g.dijkstra3()
    end = time.Since(start)
    log.Println("Dijkstra3 took %s", end)
    start = time.Now()
    g.singledijkstra()
    end = time.Since(start)
    log.Println("Single Dijkstra test took %s", end)
    // log.Println(g.DijkstraRecoveryCosts)
    // log.Println(g.DijkstraRecoveryCosts2)
    // log.Println(g.DijkstraRecoveryCosts3)
}

func (g *Game) createDistanceMap(decay float64) [][][][]float64 {
    // Creates a distance map so that we can easily divide a map to get ratios that we are interested in
    // self.distance_map[x, y, :, :] returns an array of (Width, Height) that gives the distance (x, y) is from (i, j) for all i, j
    // Note that the actual distance from x, y, to i, j is set to 1 to avoid divide by zero errors. Anything that utilizes this function should be aware of this fact.

    // Create the base map for 0, 0
    zz := make([][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        zz[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            dist_x := math.Min(float64(x), float64((g.Width-x) % g.Width))
            dist_y := math.Min(float64(y), float64((g.Height-y) % g.Height))
            dist := math.Max(dist_x + dist_y, 1)
            zz[x][y] = float64(dist * math.Exp(decay))
        }
    }

    distance_map := make([][][][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        distance_map[x] = make([][][]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            distance_map[x][y] = make([][]float64, g.Width)
            for i := 0; i < g.Width; i++ {
                distance_map[x][y][i] = make([]float64, g.Height)
            }
            distance_map[x][y] = roll_xy(zz, x, y)
        }
    }
    return distance_map
}

func (g *Game) update() {
    g.updateMaps()
    g.updateStats()
    // g.updateConfigs()
}

func (g *Game) updateMaps() {
    g.updateCalcMaps()
    g.updateOwnerMaps()
    g.updateBorderMaps()
    g.updateDistanceMaps()
    // g.updateEnemyMaps()  We'll do this later...

    g.updateValueMaps()
}

func (g *Game) updateValueMaps() {

    // Calculate a recovery cost map for every neutral cell.
    g.RecoveryCostMap = make([][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        g.RecoveryCostMap[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            if g.OwnerMap[x][y] == 0 {
                g.RecoveryCostMap[x][y] = g.StrengthMap[x][y] / g.ProductionMap01[x][y]
            }
        }
    }

    g.GlobalContributionMap = make([][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        g.GlobalContributionMap[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            cellGlobalBonus := 0.0
            for i := 0; i < g.Width; i++ {
                for j := 0; j < g.Height; j++ {
                    if g.OwnerMap[i][j] == 0 {
                        cellGlobalBonus += (1 / g.RecoveryCostMap[i][j]) / (g.DijkstraRecoveryCosts[x][y][i][j])
                    }
                }
            }
            g.GlobalContributionMap[x][y] = cellGlobalBonus
        }
    }

    g.ValueMap = make([][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        g.ValueMap[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y ++ {
            g.ValueMap[x][y] = g.RecoveryCostMap[x][y] - g.GlobalContributionMap[x][y]
        }
    }
}

func (g *Game) updateCalcMaps() {
    g.StrengthMap1 = MaxAcross2d(g.StrengthMap, 1)
    g.StrengthMap01 = MaxAcross2d(g.StrengthMap, 0.1)
}

func (g *Game) updateOwnerMaps() {
    g.IsOwnedMap = make([][]float64, g.Width)
    g.IsNeutralMap = make([][]float64, g.Width)
    g.IsEnemyMap = make([][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        g.IsOwnedMap[x] = make([]float64, g.Height)
        g.IsNeutralMap[x] = make([]float64, g.Height)
        g.IsEnemyMap[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            switch g.OwnerMap[x][y]{
            case 0:
                g.IsNeutralMap[x][y] = 1
            case g.MyID:
                g.IsOwnedMap[x][y] = 1
            default:
                g.IsEnemyMap[x][y] = 1
            }
        }
    }
}

func (g *Game) updateDistanceMaps() {
    borderSquares := make([]*Square, 0, 20)
    ownedSquares := make([]*Square, 0, 20)
    combatSquares := make([]*Square, 0, 20)
    for x := 0; x < g.Width; x++ {
        for y := 0; y < g.Height; y++ {
            if g.Squares[x][y].Owner == 0 {
                isBorder := false
                for j := 0; j < 4; j++ {
                    if g.Squares[x][y].Neighbors[j].Owner == g.MyID {
                        isBorder = true
                    }
                }
                if isBorder == true {
                    borderSquares = append(borderSquares, g.Squares[x][y])
                    if g.Squares[x][y].Strength == 0 {
                       combatSquares = append(combatSquares, g.Squares[x][y])
                    }
                }

            }
            if g.Squares[x][y].Owner == g.MyID {
                ownedSquares = append(ownedSquares, g.Squares[x][y])
            }
        }
    }
    g.DistanceFromBorder = g.floodFill(borderSquares, 999, true)
    // log.Println(g.DistanceFromBorder)
    g.DistanceFromOwned = g.floodFill(ownedSquares, 999, false)
    g.DistanceFromCombat = g.floodFill(combatSquares, 999, true)
}

func (g *Game) updateBorderMaps() {
    g.BorderMap = make([][]float64, g.Width)
    g.CombatZoneMap = make([][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        g.BorderMap[x] = make([]float64, g.Height)
        g.CombatZoneMap[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            if g.Squares[x][y].Owner == 0 {
                for _, n := range g.Squares[x][y].Neighbors {
                    if n.Owner == g.MyID {
                        g.BorderMap[x][y] = 1
                        if n.Strength == 0 {
                            g.CombatZoneMap[x][y] = 1
                        }
                    }
                }
            }
        }
    }
}

func (g *Game) updateStats() {
    g.TurnsLeft = g.MaxTurns - float64(g.Frame)
    g.PercentOwned = Sum2d(g.IsOwnedMap) / float64(g.Width * g.Height)
}

func (g *Game) getMoves() {
    // Main logic controlling code
    g.attackBorders()
    g.moveInnerSquares()
    // g.eachSquareMoves()
}

func (g *Game) attackBorders() {

    // For now, a simple attack border code.
    for x := 0; x < g.Width; x++ {
        for y := 0; y < g.Height; y++ {
            if g.BorderMap[x][y] == 1 {
                square := g.Squares[x][y]
                g.attackCell(square, 1)
            }
        }
    }
}

func (g *Game) moveInnerSquares() {
    // Simple function. just move towards highest value border.

    for x := 0; x < g.Width; x++ {
        for y := 0; y < g.Height; y++ {
            square := g.Squares[x][y]
            if square.Owner == g.MyID && square.Move == -1 && square.Strength > (square.Production * g.Buildup) {
                var target *Square
                targetVal := 9999.0
                for i := 0; i < g.Width; i++ {
                    for j := 0; j < g.Height; j++ {
                        // Note, need to add some sort of distance modifier.
                        if g.BorderMap[i][j] == 1 && targetVal > g.ValueMap[i][j] {
                            targetVal = g.ValueMap[i][j]
                            target = g.Squares[i][j]
                        }
                    }
                }
                g.moveSquareToTarget(g.Squares[x][y], target, true)
            }
        }
    }
}

func (g *Game) attackCell(target *Square, maxDistance int) bool {
    // Attempts to coordinate attacks to the target Square by calling cells distance out.
    for cellsOut := 1.0; cellsOut <= float64(maxDistance); cellsOut++ {
        // Don't attempt to coordinate a multi-cell attack into a combat zone
        if cellsOut > 1 && g.CombatZoneMap[target.X][target.Y] == 1 {
            return false
        }
        movingCells := make([]*Square, 0, 5)
        stillCells := make([]*Square, 0, 5)
        targetDistMap := g.floodFill([]*Square{target}, cellsOut, true)
        availableStrength := 0.0
        availableProduction := 0.0
        stillStrength := 0.0
        for x := 0; x < g.Width; x++ {
            for y := 0; y < g.Height; y++ {
                if targetDistMap[x][y] == -1 {
                    targetDistMap[x][y] = 0
                } else if g.IsOwnedMap[x][y] != 1 || g.MoveMap[x][y] != -1 {
                    targetDistMap[x][y] = 0
                } else if targetDistMap[x][y] > 0 {
                    availableStrength += g.StrengthMap[x][y]
                    availableProduction += (cellsOut - targetDistMap[x][y]) * g.ProductionMap[x][y]
                    if (cellsOut - targetDistMap[x][y]) > 0 {
                        stillCells = append(stillCells, g.Squares[x][y])
                        stillStrength += g.StrengthMap[x][y]
                    } else {
                        movingCells = append(movingCells, g.Squares[x][y])
                    }
                }
            }
        }

        if (availableProduction + availableStrength) > target.Strength {
            for _, square := range stillCells {
                g.makeMove(square, 4)
            }
            neededStrengthFromMovers := target.Strength - availableProduction - stillStrength + 1

            if neededStrengthFromMovers > 0 {
                sort.Sort(sort.Reverse(ByStrength(movingCells)))
                for _, square := range movingCells {
                    if square.Strength > 0 {
                        if cellsOut == 1 {
                            g.moveSquareToTarget(square, target, false)
                        } else {
                            g.moveSquareToTarget(square, target, true)
                        }
                        neededStrengthFromMovers -= square.Strength
                        if neededStrengthFromMovers < 0 {
                            break
                        }
                    }
                }
            }
            return true
        } else {
            cellsOut += 1
        }
    }
    return false
}

func (g *Game) moveSquareToTarget(s *Square, d *Square, throughFriendly bool) bool {
    // Does a "simple" movement based on a BFS.
    distanceMap := g.floodFillToTarget(s, d, throughFriendly)
    sDist := distanceMap[s.X][s.Y]
    if sDist == -1 || sDist == 0 {
        // Couldn't find a path to the destination or trying to move STILL
        return false
    }

    pathChoices := make([]int, 0, 2)
    for dir := 0; dir < 4; dir++ {
        n := s.Neighbors[dir]
        if distanceMap[n.X][n.Y] == (sDist - 1) {
            pathChoices = append(pathChoices, dir)
        }
    }

    // There should be at most 2 cells in pathChoices.
    // We can do a sort by production here if we think it's valuable. I don't feel like writing the sort function right now.

    g.makeMove(s, pathChoices[0])
    return true

    // Strength collision code goes here.
}


func (g *Game) eachSquareMoves() {
    // Each square decides on their own whether or not to move.
    // For now, let's just loop through the list of squares to determine who moves
    for x := 0; x < g.Width; x++ {
        for y := 0; y < g.Height; y++ {
            square := (g.Squares[x][y])
            if square.Owner == g.MyID && square.Move == -1 {
                // Check distance from border
                if g.DistanceFromBorder[x][y] == 1.0 {
                    // We're at a border, check if we can attack a cell
                    for d, n := range square.Neighbors {
                        if n.Owner != g.MyID && square.Strength > n.Strength {
                            g.makeMove(square, d)
                            break
                        }
                    }
                }
            }
        }
    }
}

func (g *Game) makeMove(square *Square, d int) {
    g.MoveMap[square.X][square.Y] = float64(d)

    if d == -1 {
        // Reset the square move
        if square.Target != nil {
            delete(square.Target.MovingHere, square.Vertex)
            square.Target = nil
        }
        square.Move = -1
        return
    }
    if square.Move != -1 {
        if square.Target != nil {
            delete(square.Target.MovingHere, square.Vertex)
            square.Target = nil
        }
    }
    square.Move = d
    // log.Println(d)
    if d != 4 {
        square.Target = square.Neighbors[d]
        square.Target.MovingHere[square.Vertex] = square
    }
}

func (g *Game) floodFill(sources []*Square, maxDistance float64, friendly_only bool) [][]float64 {
    // Returns a [][]int that contains the distance to the source through friendly squares only.
    // -1 : Non friendly spaces or friendly spaces unreachable to sources through friendly squares
    // 0 : Source squares
    // >0 : Friendly square distance to closest source square.
    q := sources
    distanceMap := make([][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        distanceMap[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            distanceMap[x][y] = -1
        }
    }

    if len(sources) == 0 {
        return distanceMap
    }

    // Set all source squares to 0
    for _, square := range q {
        distanceMap[square.X][square.Y] = 0
    }

    for ; len(q) > 0 ; {
        c := q[0]
        q = q[1:]
        currentDistance := distanceMap[c.X][c.Y]
        for _, n := range c.Neighbors {
            if (distanceMap[n.X][n.Y] == -1 || distanceMap[n.X][n.Y] > (currentDistance + 1)) {
                if (friendly_only && n.Owner == g.MyID) || (!friendly_only) {
                    distanceMap[n.X][n.Y] = currentDistance + 1
                    if currentDistance < maxDistance - 1 {
                        q = append(q, n)
                    }
                }
            }
        }
    }
    return distanceMap
}

func (g *Game) floodFillToTarget(source *Square, destination *Square, friendly_only bool) [][]float64 {
    // We start the fill AT the destination so we can get # of squares from source to destination.
    q := make([]*Square, 1, 5)
    q[0] = destination
    distanceMap := make([][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        distanceMap[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            distanceMap[x][y] = -1
        }
    }
    distanceMap[destination.X][destination.Y] = 0
    for ; len(q) > 0 && distanceMap[source.X][source.Y] == -1; {
        c := q[0]
        q = q[1:]
        currentDistance := distanceMap[c.X][c.Y]
        for _, n := range c.Neighbors {
            if (distanceMap[n.X][n.Y] == -1 || distanceMap[n.X][n.Y] > (currentDistance + 1)) {
                if (friendly_only && n.Owner == g.MyID) || (!friendly_only) {
                    distanceMap[n.X][n.Y] = currentDistance + 1
                    q = append(q, n)
                }
            }
        }
    }
    return distanceMap
}

func roll_xy(mat [][]float64, ox int, oy int ) [][]float64 {
    // Offsets the map in the x axis by the # of spaces.
    x_len := len(mat)
    newMatrix := make([][]float64, x_len)
    for x := 0; x < x_len; x++ {
        y_len := len(mat[x])
        newMatrix[x] = make([]float64, y_len)
        for y := 0; y < y_len; y++ {
            newMatrix[x][y] = mat[(x - ox + x_len) % x_len][(y - oy + y_len) % y_len]
        }
    }
    return newMatrix
}

func roll_x(mat [][]float64, offset int) [][]float64 {
    // Offsets the map in the x axis by the # of spaces.
    x_len := len(mat)
    newMatrix := make([][]float64, x_len)
    for x := 0; x < x_len; x++ {
        y_len := len(mat[x])
        newMatrix[x] = make([]float64, y_len)
        for y := 0; y < y_len; y++ {
            newMatrix[x][y] = mat[(x - offset) % x_len][y]
        }
    }
    return newMatrix
}

func roll_y(mat [][]float64, offset int) [][]float64 {
    // Offsets the map in the x axis by the # of spaces.
    x_len := len(mat)
    newMatrix := make([][]float64, x_len)
    for x := 0; x < x_len; x++ {
        y_len := len(mat[x])
        newMatrix[x] = make([]float64, y_len)
        for y := 0; y < y_len; y++ {
            newMatrix[x][y] = mat[x][(y - offset) % y_len]
        }
    }
    return newMatrix
}

func (s *Square) afterInitUpdate() {
    // Should only be called after all squares are initialized
    s.North = s.Game.Squares[s.X][(s.Y - 1 + s.Height) % s.Height]
    s.East = s.Game.Squares[(s.X + 1 + s.Width) % s.Width][s.Y]
    s.South = s.Game.Squares[s.X][(s.Y + 1 + s.Height) % s.Height]
    s.West = s.Game.Squares[(s.X - 1 + s.Width) % s.Width][s.Y]
    s.Neighbors = []*Square{s.North, s.East, s.South, s.West}  // doesn't include self.
    s.ResetStatus = true
    s.MovingHere = make(map[int]*Square)
    s.Target = nil
}

func (s *Square) update(owner float64, strength float64) {
    s.Owner = owner
    s.Strength = strength
    s.resetMove()
}

func (s *Square) resetMove() {
    s.Move = -1
    s.ResetStatus = true
    s.MovingHere = make(map[int]*Square)
    s.Target = nil
}

func (g *Game) sendFrame() {
    var outString string
    for _, squareX := range g.Squares {
        for _, square := range squareX {
            if square.Owner == g.MyID {
                if square.Strength == 0 {  // Squares with 0 strength shouldn't move
                    square.Move = 4
                }
                if square.Move == -1 {
                    // If we didn't actually assign a move, make sure it's still coded to STILL
                    square.Move = 4
                }
                outString = fmt.Sprintf("%s %d %d %d", outString, square.X, square.Y, translate_cardinal(square.Move))
            }
        }
    }
    fmt.Println(outString)
}

func (g *Game) deserializeMapSize() {
    splitString := strings.Split(g.getString(), " ")
    g.Width, splitString = int_str_array_pop(splitString)
    g.Height, splitString = int_str_array_pop(splitString)
}

func (g *Game) deserializeProductions() {
    splitString := strings.Split(g.getString(), " ")

    g.ProductionMap = make([][]float64, g.Width)
    for x := 0; x < g.Width; x++ {
        g.ProductionMap[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            var prod_val int
            prod_val, splitString = int_str_array_pop(splitString)
            g.ProductionMap[x][y] = float64(prod_val)
        }
    }
}

func (g *Game) getString() string {
    retstr, _ := g.Reader.ReadString('\n')
    retstr = strings.TrimSpace(retstr)
    return retstr
}

func (g *Game) getInt() int {
    i, err := strconv.Atoi(g.getString())
    if err != nil {
        log.Printf("Whoopse", err)
    }
    return i
}

func int_str_array_pop(input []string) (int, []string) {
    ret, err := strconv.Atoi(input[0])
    input = input[1:]
    if err != nil {
        log.Printf("Whoopse", err)
    }
    return ret, input
}

func translate_cardinal(d int) int {
    // Cardinal index used by the framework is:
    // NORTH = 0, EAST = 1, SOUTH = 2, WEST = 3, STILL = 4
    // Cardinal index used by the game is:
    // STILL = 0, NORTH = 1, EAST = 2, SOUTH = 3, WEST = 4
    return (d + 1) % 5
}

func getOffset(d Direction) (int, int) {
    switch d {
    case NORTH:
        return 0, -1
    case EAST:
        return 1, 0
    case SOUTH:
        return 0, 1
    case WEST:
        return -1, 0
    case STILL:
        return 0, 0
    }
    return 999, 999
}

func oppositeDirection(d Direction) Direction {
    if d == STILL {
        return STILL
    }
    return (d + 2) % 4
}

func Max2d(array [][]float64) float64 {
    // Takes a 2d slice and returns the max value
    max := array[0][0]
    for x := 0; x < len(array); x++ {
        for y := 0; y < len(array[x]); y++ {
            if array[x][y] > max {
                max = array[x][y]
            }
        }
    }
    return max
}

func Min2d(array [][]float64) float64 {
    // Takes a 2d slice and returns the min value
    min := array[0][0]
    for x := 0; x < len(array); x++ {
        for y := 0; y < len(array[x]); y++ {
            if array[x][y] < min {
                min = array[x][y]
            }
        }
    }
    return min
}

func MinAcross2d(array [][]float64, minVal float64) [][]float64 {
    // Returns an array which does a piecewise min between an array and a float64
    retArray := make([][]float64, len(array))
    for x:= 0; x < len(array); x++ {
        retArray[x] = make([]float64, len(array[x]))
        for y:= 0; y < len(array[x]); y++ {
            retArray[x][y] = math.Min(array[x][y], minVal)
        }
    }
    return retArray
}

func MaxAcross2d(array [][]float64, maxVal float64) [][]float64 {
    // Returns an array which does a piecewise max between an array and a float64
    retArray := make([][]float64, len(array))
    for x:= 0; x < len(array); x++ {
        retArray[x] = make([]float64, len(array[x]))
        for y:= 0; y < len(array[x]); y++ {
            retArray[x][y] = math.Max(array[x][y], maxVal)
        }
    }
    return retArray
}

func Sum2d(array [][]float64) float64 {
    // Takes a 2d array and returns the sum of all values
    val := 0.0
    for x := 0; x < len(array); x++ {
        for y := 0; y < len(array[x]); y++ {
            val += float64(array[x][y])
        }
    }
    return val
}

func main() {
    f, _ := os.OpenFile("gologfile.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    log.SetOutput(f)
    log.Println("Hi")
    game := NewGame()
    for {
        game.getFrame()
        game.update()

        game.getMoves()

        game.sendFrame()
    }
}



func (g *Game) getxy(vertex int) (int, int) {
    x := int(math.Floor(float64(vertex) / float64(g.Height)))
    y := vertex % g.Height
    return x, y
}

func getxy(vertex int) (int, int) {
    x := int(math.Floor(float64(vertex) / float64(Height)))
    y := vertex % Height
    return x, y
}


func (g *Game) createDijkstraMaps() {
    // Creates the dijkstra map(s) that will be utilized in this bot.

    // A 4-d array is created which contains all the information on the costs and routes for every cell to every other cell.
    // Ignores who owns the cell

    // Run Dijkstra on recovery cost for all squares to all squares.

    // edges, nodes := g.makeGraphRecovery()
    _, nodes := g.makeGraphRecovery()
    g.DijkstraRecoveryCosts = make([][][][]float64, g.Width)
    g.DijkstraRecoveryPaths = make([][][][]int, g.Width)
    for x := 0; x < g.Width; x++ {
        g.DijkstraRecoveryCosts[x] = make([][][]float64, g.Height)
        g.DijkstraRecoveryPaths[x] = make([][][]int, g.Height)
        for y := 0; y < g.Height; y++ {
            g.DijkstraRecoveryCosts[x][y] = make([][]float64, g.Width)
            g.DijkstraRecoveryPaths[x][y] = make([][]int, g.Width)
            for a := 0; a < g.Width; a++ {
                g.DijkstraRecoveryCosts[x][y][a] = make([]float64, g.Height)
                g.DijkstraRecoveryPaths[x][y][a] = make([]int, g.Height)
            }
        }
    }
    for startV := 0; startV < g.Height * g.Width; startV++ {
        vx, vy := g.getxy(startV)
        pathList := dijkstra(nodes, nodes[startV], nil)
        for _, path := range pathList{
            target := path.targetVertex
            tx, ty := g.getxy(target)
            g.DijkstraRecoveryCosts[vx][vy][tx][ty] = path.length
            g.DijkstraRecoveryPaths[vx][vy][tx][ty] = path.path
            // if len(path.path) > 1 {
            //     g.DijkstraRecoveryPaths[vx][vy][tx][ty] = path.path[1]
            // } else {
            //     g.DijkstraRecoveryPaths[vx][vy][tx][ty] = -9999
            // }
        }
    }
}

func (g *Game) singledijkstra() {
    // Creates the dijkstra map(s) that will be utilized in this bot.

    // A 4-d array is created which contains all the information on the costs and routes for every cell to every other cell.
    // Ignores who owns the cell

    // Run Dijkstra on recovery cost for all squares to all squares.

    // edges, nodes := g.makeGraphRecovery()
    _, nodes := g.makeGraphRecovery()
    g.DijkstraRecoveryCosts = make([][][][]float64, g.Width)
    g.DijkstraRecoveryPaths = make([][][][]int, g.Width)
    for x := 0; x < g.Width; x++ {
        g.DijkstraRecoveryCosts[x] = make([][][]float64, g.Height)
        g.DijkstraRecoveryPaths[x] = make([][][]int, g.Height)
        for y := 0; y < g.Height; y++ {
            g.DijkstraRecoveryCosts[x][y] = make([][]float64, g.Width)
            g.DijkstraRecoveryPaths[x][y] = make([][]int, g.Width)
            for a := 0; a < g.Width; a++ {
                g.DijkstraRecoveryCosts[x][y][a] = make([]float64, g.Height)
                g.DijkstraRecoveryPaths[x][y][a] = make([]int, g.Height)
            }
        }
    }
    for startV := 0; startV < 1; startV++ {
        vx, vy := g.getxy(startV)
        pathList := dijkstra(nodes, nodes[startV], nil)
        for _, path := range pathList{
            target := path.targetVertex
            tx, ty := g.getxy(target)
            g.DijkstraRecoveryCosts[vx][vy][tx][ty] = path.length
            g.DijkstraRecoveryPaths[vx][vy][tx][ty] = path.path
            // if len(path.path) > 1 {
            //     g.DijkstraRecoveryPaths[vx][vy][tx][ty] = path.path[1]
            // } else {
            //     g.DijkstraRecoveryPaths[vx][vy][tx][ty] = -9999
            // }
        }
    }
}


type Edge struct {
    V1, V2 int
    Dist float64
}

type Node struct {
    V int
    TDist float64  // tentative distance
    Prev *Node
    Done bool  // True when Tdist and Prev represent the shortest path
    Neighbors []Neighbor  // Edges from this vertex
    Rx int  // heap.Remove index
}

type Neighbor struct {
    Node *Node // Node corresponding to a vertex
    Dist float64  // Distance to this node, from whatever node references this
}

func (g *Game) makeGraphRecovery() ([]Edge, []*Node) {
    // Creates a graph from the Squares object to be used for Dijkstra's
    // To hopefully reduce # of calls, this actually builds the graph the opposite way. Builds the graph from neighbors instead of TO neighbors.
    graph := make([]Edge, 0, int(g.Width * g.Height * 4))
    nodes := make([]*Node, int(g.Width * g.Height))
    for x := 0; x < g.Width; x++ {
        for y := 0; y < g.Height; y++ {
            // Builds edges based on how much it costs to get to THIS cell FROM neighbors.
            graph = append(graph, Edge{V1: g.Squares[x][y].North.Vertex, V2: g.Squares[x][y].Vertex, Dist: g.StrengthMap1[x][y] / g.ProductionMap01[x][y]})
            graph = append(graph, Edge{V1: g.Squares[x][y].South.Vertex, V2: g.Squares[x][y].Vertex, Dist: g.StrengthMap1[x][y] / g.ProductionMap01[x][y]})
            graph = append(graph, Edge{V1: g.Squares[x][y].West.Vertex, V2: g.Squares[x][y].Vertex, Dist: g.StrengthMap1[x][y] / g.ProductionMap01[x][y]})
            graph = append(graph, Edge{V1: g.Squares[x][y].East.Vertex, V2: g.Squares[x][y].Vertex, Dist: g.StrengthMap1[x][y] / g.ProductionMap01[x][y]})

            // Add the node
            nodes[g.Squares[x][y].Vertex] = &Node{V: g.Squares[x][y].Vertex}
        }
    }

    for _, e := range graph {
        n1 := nodes[e.V1]
        n2 := nodes[e.V2]
        n1.Neighbors = append(n1.Neighbors, Neighbor{n2, e.Dist})
    }
    return graph, nodes
}

type path struct {
    // path []int
    path int
    length float64
    targetVertex int
}

func dijkstra(allNodes []*Node, startNode, endNode *Node) (pathList []path) {
    // 1. Assign to every node a tentative distance value: set it to zero for our initial node and to infinity for all other nodes.
    // 2. Set the initial node as current. Mark all other nodes unvisited. Create a set of all the unvisited nodes called the unvisited set.
    for _, nd := range allNodes {
        nd.TDist = math.Inf(1)
        nd.Done = false
        nd.Prev = nil
        nd.Rx = -1
    }

    current := startNode
    current.TDist = 0
    var unvisited ndList
    for {
        // 3. For the current node, consider all of its unvisited neighbors and calculate their tentative distances. Compare the newly calculated tentative distance to the current assigned value and assign the smaller one. For example, if the current node A is marked with a distance of 6, and the edge connecting it with a neighbor B has length 2, then the distance to B (through A) will be 6 + 2 = 8. If B was previously marked with a distance greater than 8 then change it to 8. Otherwise, keep the current value.
        for _, nb := range current.Neighbors {
            if nd := nb.Node; !nd.Done {
                if d := current.TDist + nb.Dist; d < nd.TDist {
                    nd.TDist = d
                    nd.Prev = current
                    if nd.Rx < 0 {
                        heap.Push(&unvisited, nd)
                    } else {
                        heap.Fix(&unvisited, nd.Rx)
                    }
                }
            }
        }
        // 4. When we are done considering all of the neighbors of the current node, mark the current node as visited and remove it from the unvisited set. A visited node will never be checked again.
        current.Done = true
        if endNode == nil || current == endNode {
            // Record path and distance for return value
            distance := current.TDist
            // Recover path by tracing prev links
            var p []int
            target := current.V
            for ; current != nil; current = current.Prev {
                p = append(p, current.V)
            }
            // Reverse the list
            // for i := (len(p) + 1) / 2; i > 0; i-- {
            //     p[i-1], p[len(p)-i] = p[len(p)-1], p[i-1]
            // }
            // pathList = append(pathList, path{p, distance, target})
            var prior int
            if len(p) > 1 {
                prior = p[len(p) - 1]
            } else {
                prior = -9999
            }

            pathList = append(pathList, path{prior, distance, target})
            // 5. If the destination node has been marked visited (when planning a route between two specific nodes) or if the smallest tentative distance among the nodes in the unvisited set is infinity (when planning a complete traversal; occurs when there is no connection between the initial node and remaining unvisited nodes), then stop. The algorithm has finished.
            if endNode != nil {
                return
            }
        }
        if len(unvisited) == 0 {
            break  // No more reachable nodes
        }
        // 6. Otherwise, select the unvisited node that is marked with the smallest tentative distance, set it as the new "current node", and go back to step 3.
        current = heap.Pop(&unvisited).(*Node)
    }
    return
}


// ndList implements a container/heap
type ndList []*Node

func (n ndList) Len() int {
    return len(n)
}

func (n ndList) Less(i, j int) bool {
    return n[i].TDist < n[j].TDist
}

func (n ndList) Swap(i, j int) {
    n[i], n[j] = n[j], n[i]
    n[i].Rx = i
    n[j].Rx = j
}

func (n *ndList) Push(x interface{}) {
    nd := x.(*Node)
    nd.Rx = len(*n)
    *n = append(*n, nd)
}

func (n *ndList) Pop() interface{} {
    s := *n
    last := len(s) - 1
    r := s[last]
    *n = s[:last]
    r.Rx = -1
    return r
}


type ByStrength []*Square

func (s ByStrength) Len() int {
    return len(s)
}

func (s ByStrength) Swap(i, j int) {
    s[i], s[j] = s[j], s[i]
}

func (s ByStrength) Less(i, j int) bool {
    return s[i].Strength < s[j].Strength
}

func (g *Game) createDijkstraMaps2() {
    // Creates the dijkstra map(s) that will be utilized in this bot.

    // A 4-d array is created which contains all the information on the costs and routes for every cell to every other cell.
    // Ignores who owns the cell

    // Run Dijkstra on recovery cost for all squares to all squares.

    // edges, nodes := g.makeGraphRecovery()
    g.DijkstraRecoveryCosts2 = make([][][][]float64, g.Width)
    g.DijkstraRecoveryPaths2 = make([][][][]int, g.Width)
    for x := 0; x < g.Width; x++ {
        g.DijkstraRecoveryCosts2[x] = make([][][]float64, g.Height)
        g.DijkstraRecoveryPaths2[x] = make([][][]int, g.Height)
        for y := 0; y < g.Height; y++ {
            g.DijkstraRecoveryCosts2[x][y] = make([][]float64, g.Width)
            g.DijkstraRecoveryPaths2[x][y] = make([][]int, g.Width)
            for a := 0; a < g.Width; a++ {
                g.DijkstraRecoveryCosts2[x][y][a] = make([]float64, g.Height)
                g.DijkstraRecoveryPaths2[x][y][a] = make([]int, g.Height)
            }
        }
    }
    for startV := 0; startV < g.Height * g.Width; startV++ {
        vx, vy := g.getxy(startV)
        costs, paths := g.dijkstra2(startV, -1)

        for x := 0; x < g.Width; x++ {
            for y := 0; y < g.Height; y++ {
                g.DijkstraRecoveryCosts2[vx][vy][x][y] = costs[x][y]
                g.DijkstraRecoveryPaths2[vx][vy][x][y] = paths[x][y]
            }
        }
    }
}

func (g *Game) dijkstra2(startNode int, endNode int) ([][]float64, [][]int){

    TDist := make([][]float64, g.Width)
    Done := make([][]bool, g.Width)
    Prev := make([][]int, g.Width) // Vertex # will be used here.
    Rx = make([][]int, g.Width)
    Dist := make([][]float64, g.Width)

    for x := 0; x < g.Width; x++ {
        TDist[x] = make([]float64, g.Height)
        Done[x] = make([]bool, g.Height)
        Prev[x] = make([]int, g.Height)
        Rx[x] = make([]int, g.Height)
        Dist[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            TDist[x][y] = math.Inf(1)
            Done[x][y] = false
            Prev[x][y] = -1
            Rx[x][y] = -1
            Dist[x][y] = math.Inf(1)
        }
    }

    current := startNode
    cx, cy := g.getxy(current)
    TDist[cx][cy] = 0

    var unvisited intList
    for {
        // log.Println("start")
        // log.Println(startNode)
        // log.Println(Rx)
        // 3. For the current node, consider all of its unvisited neighbors and calculate their tentative distances. Compare the newly calculated tentative distance to the current assigned value and assign the smaller one. For example, if the current node A is marked with a distance of 6, and the edge connecting it with a neighbor B has length 2, then the distance to B (through A) will be 6 + 2 = 8. If B was previously marked with a distance greater than 8 then change it to 8. Otherwise, keep the current value.

        // North
        nx, ny := cx, (cy - 1 + g.Height) % g.Height
        // log.Println("North")
        // log.Println(nx)
        // log.Println(ny)
        if !Done[nx][ny] {
            // log.Println("done")
            cost := g.StrengthMap1[nx][ny]/g.ProductionMap01[nx][ny]
            // log.Println("cost")
            nDist := TDist[cx][cy] + cost
            // log.Println("ndist")
            if nDist < TDist[nx][ny] {
                TDist[nx][ny] = nDist
                // log.Println("tdist")
                Prev[nx][ny] = current
                // log.Println("prev")
                if Rx[nx][ny] < 0 {
                    // log.Println("prepush")
                    heapValue := float64(nx * g.Height + ny) * 1000000000.0 + float64(len(unvisited)) * 100000.0 + float64(TDist[nx][ny]) / 1000
                    Rx[nx][ny] = len(unvisited)
                    // log.Println("rx")
                    // log.Println(heapValue)
                    heap.Push(&unvisited, heapValue)
                    // log.Println("push")
                } else {
                    // log.Println("prefix")
                    heap.Fix(&unvisited, Rx[nx][ny])
                    // log.Println("fix")
                }
            }
        }
        // South
        nx, ny = cx, (cy + 1) % g.Height
        // log.Println("South")
        // log.Println(nx)
        // log.Println(ny)

        if !Done[nx][ny] {
            // log.Println("done")
            cost := g.StrengthMap1[nx][ny]/g.ProductionMap01[nx][ny]
            // log.Println("cost")
            nDist := TDist[cx][cy] + cost
            // log.Println("ndist")
            if nDist < TDist[nx][ny] {
                TDist[nx][ny] = nDist
                // log.Println("tdist")
                Prev[nx][ny] = current
                // log.Println("prev")
                if Rx[nx][ny] < 0 {
                    // log.Println("prepush")
                    heapValue := float64(nx * g.Height + ny) * 1000000000.0 + float64(len(unvisited)) * 100000.0 + float64(TDist[nx][ny]) / 1000
                    Rx[nx][ny] = len(unvisited)
                    // log.Println("rx")
                    // log.Println(heapValue)
                    heap.Push(&unvisited, heapValue)
                    // log.Println("push")
                } else {
                    // log.Println("prefix")
                    heap.Fix(&unvisited, Rx[nx][ny])
                    // log.Println("fix")
                }
            }
        }
        // West
        nx, ny = (cx - 1 + g.Width) % g.Width, cy
        // log.Println("West")
        // log.Println(nx)
        // log.Println(ny)
        if !Done[nx][ny] {
            cost := g.StrengthMap1[nx][ny]/g.ProductionMap01[nx][ny]
            nDist := TDist[cx][cy] + cost
            if nDist < TDist[nx][ny] {
                TDist[nx][ny] = nDist
                Prev[nx][ny] = current
                if Rx[nx][ny] < 0 {
                    heapValue := float64(nx * g.Height + ny) * 1000000000.0 + float64(len(unvisited)) * 100000.0 + float64(TDist[nx][ny]) / 1000
                    Rx[nx][ny] = len(unvisited)
                    heap.Push(&unvisited, heapValue)
                } else {
                    heap.Fix(&unvisited, Rx[nx][ny])
                }
            }
        }
        // East
        nx, ny = (cx + 1) % g.Width, cy
        // log.Println("East")
        // log.Println(nx)
        // log.Println(ny)
        if !Done[nx][ny] {
            cost := g.StrengthMap1[nx][ny]/g.ProductionMap01[nx][ny]
            nDist := TDist[cx][cy] + cost
            if nDist < TDist[nx][ny] {
                TDist[nx][ny] = nDist
                Prev[nx][ny] = current
                if Rx[nx][ny] < 0 {
                    heapValue := float64(nx * g.Height + ny) * 1000000000.0 + float64(len(unvisited)) * 100000.0 + float64(TDist[nx][ny]) / 1000
                    Rx[nx][ny] = len(unvisited)
                    heap.Push(&unvisited, heapValue)
                } else {
                    heap.Fix(&unvisited, Rx[nx][ny])
                }
            }
        }
        // 4. When we are done considering all of the neighbors of the current node, mark the current node as visited and remove it from the unvisited set. A visited node will never be checked again.
        Done[cx][cy] = true
        if endNode == -1 || current == endNode {
            // Record path and distance for return value
            Dist[cx][cy] = TDist[cx][cy]
            // 5. If the destination node has been marked visited (when planning a route between two specific nodes) or if the smallest tentative distance among the nodes in the unvisited set is infinity (when planning a complete traversal; occurs when there is no connection between the initial node and remaining unvisited nodes), then stop. The algorithm has finished.
            if endNode != -1 {
                return Dist, Prev
            }
        }
        if len(unvisited) == 0 {
            break  // No more reachable nodes
        }
        // 6. Otherwise, select the unvisited node that is marked with the smallest tentative distance, set it as the new "current node", and go back to step 3.
        // log.Println("popping")
        next := heap.Pop(&unvisited).(float64)
        // log.Println("afterpop")
        current = int(int(next) / 1000000000)
        // log.Println("current/cx/cy")
        // log.Println(current)
        cx, cy = g.getxy(current)
        // log.Println(cx)
        // log.Println(cy)
        Rx[cx][cy] = -1
    }

    return Dist, Prev
}

// ndList implements a container/heap
type intList []float64

func (n intList) Len() int {
    return len(n)
}

func (n intList) Less(i, j int) bool {
    ni := n[i] - float64(int(n[i] / 100000.0) * 100000)
    nj := n[j] - float64(int(n[j] / 100000.0) * 100000)
    return ni < nj
}

func (n intList) Swap(i, j int) {
    currenti := int(int(n[i]) / 1000000000)
    ix, iy := getxy(currenti)
    currentj := int(int(n[j]) / 1000000000)
    jx, jy := getxy(currentj)
    // rxi := int(int(n[i]) % 1000000000 / 100000)
    // rxj := int(int(n[j]) % 1000000000 / 100000)
    n[i], n[j] = n[j], n[i]
    Rx[jx][jy] = i
    Rx[ix][iy] = j
    // n[i] += float64(i - rxj) * 100000.0
    // n[j] += float64(j - rxi) * 100000.0
}

func (n *intList) Push(x interface{}) {
    nd := x.(float64)
    *n = append(*n, nd)
}

func (n *intList) Pop() interface{} {
    s := *n
    last := len(s) - 1
    r := s[last]
    *n = s[:last]
    return r
}

func (g *Game) dijkstra3() {
    // init arrays
    endNode := -1
    g.DijkstraRecoveryCosts3 = make([][][][]float64, g.Width)
    g.DijkstraRecoveryPaths3 = make([][][][]int, g.Width)
    g.DijkstraRecoveryDone3 = make([][][][]bool, g.Width)
    TDist := make([][]float64, g.Width)
    // Dist := make([][]float64, g.Width)
    Prev := make([][]int, g.Width)
    Rx = make([][]int, g.Width)
    costMap := make([][]float64, g.Width)
    nodes := make([]*Node2, g.Width * g.Height)
    for x := 0; x < g.Width; x++ {
        g.DijkstraRecoveryCosts3[x] = make([][][]float64, g.Height)
        g.DijkstraRecoveryPaths3[x] = make([][][]int, g.Height)
        g.DijkstraRecoveryDone3[x] = make([][][]bool, g.Width)
        TDist[x] = make([]float64, g.Height)
        // Dist[x] = make([]float64, g.Height)
        Prev[x] = make([]int, g.Height)
        Rx[x] = make([]int, g.Height)
        costMap[x] = make([]float64, g.Height)
        for y := 0; y < g.Height; y++ {
            g.DijkstraRecoveryCosts3[x][y] = make([][]float64, g.Width)
            g.DijkstraRecoveryPaths3[x][y] = make([][]int, g.Width)
            g.DijkstraRecoveryDone3[x][y] = make([][]bool, g.Width)
            costMap[x][y] = g.StrengthMap1[x][y] / g.ProductionMap01[x][y]
            nodes[x * g.Height + y] = &Node2{}
            for a := 0; a < g.Width; a++ {
                g.DijkstraRecoveryCosts3[x][y][a] = make([]float64, g.Height)
                g.DijkstraRecoveryPaths3[x][y][a] = make([]int, g.Height)
                g.DijkstraRecoveryDone3[x][y][a] = make([]bool, g.Width)
                for b := 0; b < g.Height; b++ {
                    g.DijkstraRecoveryCosts3[x][y][a][b] = math.Inf(1)
                }
            }
        }
    }
    // log.Println("initdone")
    for startNode := 0; startNode < g.Width * g.Height; startNode ++ {
        log.Println("startNode")
        log.Println(startNode)

        vx, vy := g.getxy(startNode)

        var unvisited nd2List
        var n *Node2
        // Reset values
        for x := 0; x < g.Width; x++ {
            for y := 0; y < g.Height; y++ {
                n = nodes[x * g.Height + y]
                n.V = x * g.Height + y
                n.TDist = math.Inf(1)
                n.Rx = -1

                // TDist[x][y] = math.Inf(1)
                Prev[x][y] = -1
                // Rx[x][y] = -1
                // Dist[x][y] = math.Inf(1)

                if g.DijkstraRecoveryDone3[vx][vy][x][y] {
                    // We've explored this cell before. set cost and add to unvisited
                    nodes[x * g.Height + y].TDist = g.DijkstraRecoveryCosts3[vx][vy][x][y]
                    // log.Println("rx")
                    // log.Println(heapValue)
                    heap.Push(&unvisited, nodes[x * g.Height + y])
                }
            }
        }
        // log.Println("reset")
        var neighbor *Node2
        var cost, nDist float64
        current := nodes[startNode]
        cx, cy := g.getxy(current.V)
        current.TDist = 0
        // log.Println("getcurrent")

        for {
            log.Println("current.v")
            log.Println(current.V)
            log.Println("cur Dist")
            log.Println(current.TDist)
            // log.Println("start")
            // log.Println(startNode)
            // log.Println(Rx)
            // 3. For the current node, consider all of its unvisited neighbors and calculate their tentative distances. Compare the newly calculated tentative distance to the current assigned value and assign the smaller one. For example, if the current node A is marked with a distance of 6, and the edge connecting it with a neighbor B has length 2, then the distance to B (through A) will be 6 + 2 = 8. If B was previously marked with a distance greater than 8 then change it to 8. Otherwise, keep the current value.

            // North
            nx, ny := cx, (cy - 1 + g.Height) % g.Height
            neighbor = nodes[nx * g.Height + ny]
            // log.Println("North")
            // log.Println(nx)
            // log.Println(ny)
            if !g.DijkstraRecoveryDone3[vx][vy][nx][ny] {
                // log.Println("done")
                cost = costMap[nx][ny]
                // log.Println("cost")
                nDist = current.TDist + cost
                log.Println("n nDist")
                log.Println(nDist)
                log.Println("north neigh nDist")
                log.Println(neighbor.TDist)
                // log.Println("ndist")
                if nDist < neighbor.TDist {
                    neighbor.TDist = nDist
                    // log.Println("tdist")
                    Prev[nx][ny] = current.V
                    // log.Println("prev")
                    log.Println("north nrx")
                    log.Println(neighbor.Rx)
                    if neighbor.Rx < 0 {
                        log.Println("prepush")
                        heap.Push(&unvisited, neighbor)
                        // log.Println("push")
                    } else {
                        // log.Println("prefix")
                        heap.Fix(&unvisited, neighbor.Rx)
                        // log.Println("fix")
                    }
                }
            }
            // South
            nx, ny = cx, (cy + 1) % g.Height
            neighbor = nodes[nx * g.Height + ny]
            // log.Println("South")
            // log.Println(nx)
            // log.Println(ny)
            if !g.DijkstraRecoveryDone3[vx][vy][nx][ny] {
                // log.Println("done")
                cost = costMap[nx][ny]
                // log.Println("cost")
                nDist = current.TDist + cost
                log.Println("s nDist")
                log.Println(nDist)
                log.Println("sorth neigh nDist")
                log.Println(neighbor.TDist)
                // log.Println("ndist")
                if nDist < neighbor.TDist {
                    neighbor.TDist = nDist
                    // log.Println("tdist")
                    Prev[nx][ny] = current.V
                    // log.Println("prev")
                    if neighbor.Rx < 0 {
                        // log.Println("prepush")
                        // log.Println(heapValue)
                        heap.Push(&unvisited, neighbor)
                        // log.Println("push")
                    } else {
                        // log.Println("prefix")
                        heap.Fix(&unvisited, neighbor.Rx)
                        // log.Println("fix")
                    }
                }
            }
            // West
            nx, ny = (cx - 1 + g.Width) % g.Width, cy
            neighbor = nodes[nx * g.Height + ny]
            // log.Println("West")
            // log.Println(nx)
            // log.Println(ny)
            if !g.DijkstraRecoveryDone3[vx][vy][nx][ny] {
                // log.Println("done")
                cost = costMap[nx][ny]
                // log.Println("cost")
                nDist = current.TDist + cost
                // log.Println("ndist")
                if nDist < neighbor.TDist {
                    neighbor.TDist = nDist
                    // log.Println("tdist")
                    Prev[nx][ny] = current.V
                    // log.Println("prev")
                    // log.Println("west nrx")
                    // log.Println(neighbor.Rx)
                    if neighbor.Rx < 0 {
                        // log.Println("prepush")
                        // log.Println(heapValue)
                        heap.Push(&unvisited, neighbor)
                        // log.Println("push")
                    } else {
                        // log.Println("prefix")
                        heap.Fix(&unvisited, neighbor.Rx)
                        // log.Println("fix")
                    }
                }
            }
            // East
            nx, ny = (cx + 1) % g.Width, cy
            neighbor = nodes[nx * g.Height + ny]
            // log.Println("East")
            // log.Println(nx)
            // log.Println(ny)
            if !g.DijkstraRecoveryDone3[vx][vy][nx][ny] {
                // log.Println("done")
                cost = costMap[nx][ny]
                // log.Println("cost")
                nDist = current.TDist + cost
                // log.Println("ndist")
                if nDist < neighbor.TDist {
                    neighbor.TDist = nDist
                    // log.Println("tdist")
                    Prev[nx][ny] = current.V
                    // log.Println("prev")
                    if neighbor.Rx < 0 {
                        // log.Println("prepush")
                        // log.Println(heapValue)
                        heap.Push(&unvisited, neighbor)
                        // log.Println("push")
                    } else {
                        // log.Println("prefix")
                        heap.Fix(&unvisited, neighbor.Rx)
                        // log.Println("fix")
                    }
                }
            }
            // 4. When we are done considering all of the neighbors of the current node, mark the current node as visited and remove it from the unvisited set. A visited node will never be checked again.
            g.DijkstraRecoveryDone3[vx][vy][cx][cy] = true
            if endNode == -1 { // || current == endNode {
                // Record path and distance for return value
                g.DijkstraRecoveryCosts3[vx][vy][cx][cy] = current.TDist
                // 5. If the destination node has been marked visited (when planning a route between two specific nodes) or if the smallest tentative distance among the nodes in the unvisited set is infinity (when planning a complete traversal; occurs when there is no connection between the initial node and remaining unvisited nodes), then stop. The algorithm has finished.
                if endNode != -1 {
                    continue
                }
            }
            if len(unvisited) == 0 {
                break  // No more reachable nodes
            }
            // 6. Otherwise, select the unvisited node that is marked with the smallest tentative distance, set it as the new "current node", and go back to step 3.
            // log.Println("popping")
            current = heap.Pop(&unvisited).(*Node2)
            // cx, cy = g.getxy(current.V)
            log.Println("popok")
            log.Println(current.V)
            // log.Println(cx)
            // log.Println(cy)
        }


    }

    return
}


type Node2 struct {
    V int
    TDist float64  // tentative distance
    // Prev *Node
    // Done bool  // True when Tdist and Prev represent the shortest path
    Rx int  // heap.Remove index
}


// ndList implements a container/heap
type nd2List []*Node2

func (n nd2List) Len() int {
    return len(n)
}

func (n nd2List) Less(i, j int) bool {
    return n[i].TDist < n[j].TDist
}

func (n nd2List) Swap(i, j int) {
    n[i], n[j] = n[j], n[i]
    n[i].Rx = i
    n[j].Rx = j
}

func (n *nd2List) Push(x interface{}) {
    nd := x.(*Node2)
    nd.Rx = len(*n)
    *n = append(*n, nd)
}

func (n *nd2List) Pop() interface{} {
    s := *n
    last := len(s) - 1
    r := s[last]
    *n = s[:last]
    r.Rx = -1
    return r
}
