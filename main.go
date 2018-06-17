package main

import (
	"bufio"
	"container/heap"
	"fmt"
	"math"
	"os"
)

// FactWanderers multiplier
// 1 => 137
// 2 => 105
// 3 => 220
const FactWanderers = 2

// TraversableDist how far we search available cells
// 4 => 230
// 5 => 58
// 6 => 105
const TraversableDist = 5

// RangeWanderers guard
// 5 => 191
// 6 => 58
// 7  => 14
const RangeWanderers = 7

// RangeSlashers guard
const RangeSlashers = 6

type grid [][]cell
type cell int

const (
	inputWall    = "#"
	inputSpawn   = "w"
	inputShelter = "U"
	inputEmpty   = "."
)

const (
	cellWall    = iota
	cellSpawn   = iota
	cellShelter = iota
	cellEmpty   = iota
)

type coord struct {
	x, y int
}

type minionState int

const (
	stateSpawning  minionState = 0
	stateWandering minionState = 1
	stateStalking  minionState = 2
	stateRushing   minionState = 3
	stateStunned   minionState = 4
)

type explorer struct {
	id     int
	coord  coord
	sanity int
}

type minion interface {
	getCoord() coord
}

type wanderer struct {
	id         int
	coord      coord
	state      minionState
	target     int
	recallTime int
}

type slasher struct {
	id              int
	coord           coord
	state           minionState
	target          int
	changeStateTime int
}

type spawningMinion struct {
	id        int
	coord     coord
	state     minionState
	target    int
	spawnTime int
}

func (w wanderer) getCoord() coord {
	return w.coord
}

func (s slasher) getCoord() coord {
	return s.coord
}

type loggable interface {
	String() string
}

const (
	entityTypeExplorer      = "EXPLORER"
	entityTypeWanderer      = "WANDERER"
	entityTypeEffectPlan    = "EFFECT_PLAN"
	entityTypeEffectLight   = "EFFECT_LIGHT"
	entityTypeSlasher       = "SLASHER"
	entityTypeEffectShelter = "EFFECT_SHELTER"
	entityTypeEffectYell    = "EFFECT_YELL"
)

func buildGridOfWalls(width int, height int) grid {
	res := make(grid, height)
	for i := 0; i < height; i++ {
		res[i] = make([]cell, width)
	}
	return res
}

func printGrid(g grid) {
	res := ""
	for _, line := range g {
		for _, cell := range line {
			res += cellToString(cell)
		}
		res += "\n"
	}
	log(res)
}

func log(mes ...interface{}) {
	fmt.Fprintln(os.Stderr, mes...)
}

func cellToString(c cell) string {
	switch c {
	case cellWall:
		return "#"
	case cellSpawn:
		return "w"
	case cellShelter:
		return "U"
	case cellEmpty:
		return "."
	default:
		panic("unrecognized cell " + string(c))
	}
}

func parseCell(c string) cell {
	switch c {
	case inputWall:
		return cellWall
	case inputSpawn:
		return cellSpawn
	case inputShelter:
		return cellShelter
	case inputEmpty:
		return cellEmpty
	default:
		panic("unrecognized string " + c)
	}
}

func parseGrid(scanner *bufio.Scanner, width int, height int) grid {
	res := buildGridOfWalls(width, height)
	for i := 0; i < height; i++ {
		scanner.Scan()
		line := scanner.Text()
		for j, c := range line {
			res[i][j] = parseCell(string(c))
		}
	}
	return res
}

func send(command string) {
	fmt.Println(command)
}

func sendMove(x, y int, message string) {
	send(fmt.Sprintf("MOVE %d %d %s", x, y, message))
}

func sendWait(message string) {
	send(fmt.Sprintf("WAIT %s", message))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func manhattanDist(from coord, to coord) int {
	return abs(to.x-from.x) + abs(to.y-from.y)
}

func getClosestMinionCoord(from coord, minions []minion) coord {
	if len(minions) == 0 {
		panic("cannot find closest minion if there is no minion")
	}
	bestDistance := -1
	bestCoord := coord{0, 0}
	for _, m := range minions {
		d := manhattanDist(m.getCoord(), from)
		if bestDistance == -1 || d < bestDistance {
			bestDistance = d
			bestCoord = m.getCoord()
		}
	}

	return bestCoord
}

func getEmptyCells(g grid) []coord {
	res := make([]coord, 0)
	for i, line := range g {
		for j, cell := range line {
			if cell == cellEmpty {
				res = append(res, coord{j, i})
			}
		}
	}
	return res
}

func getCloseTraversableCells(g grid, from coord, distFromMe map[coord]int) []coord {
	res := make([]coord, 0)
	for i, line := range g {
		for j, cell := range line {
			d, prs := distFromMe[coord{j, i}]
			if (isTraversable(cell)) && prs && d <= TraversableDist {
				res = append(res, coord{j, i})
			}
		}
	}
	return res
}

func getFarestCoord(minions []minion, candidates []coord) coord {
	if len(candidates) == 0 {
		panic("no candidates for farest coord")
	}
	bestIndex := -1
	bestDistance := -1
	for i, c := range candidates {
		sum := 0
		for _, m := range minions {
			sum += manhattanDist(m.getCoord(), c)
		}

		if bestDistance == -1 || sum > bestDistance {
			bestIndex = i
			bestDistance = sum
			log("Farest : ", candidates[bestIndex], " with distance ", bestDistance)
		}
	}
	return candidates[bestIndex]
}

func getAwayFromMinions(g grid, me explorer, minions []minion, distFromMe map[coord]int) coord {
	// closestMinion := getClosestMinionCoord(me.coord, minions)
	empties := getCloseTraversableCells(g, me.coord, distFromMe)
	log(fmt.Sprintf("empties: %v", empties))
	return getFarestCoord(minions, empties)
}

func getBestExplorer(me explorer, explorers []explorer) coord {
	bestIndex := -1
	bestScore := -1
	for i, e := range explorers {
		if e.id != me.id && (bestScore == -1 || e.sanity > bestScore) {
			bestIndex = i
			bestScore = e.sanity
		}
	}
	return explorers[bestIndex].coord
}

func getFrighteningMinions(me explorer, wanderers []wanderer, slashers []slasher, distFromMe map[coord]int) []minion {
	minions := make([]minion, 0)

	for _, w := range wanderers {
		if d, p := distFromMe[w.coord]; p && d <= RangeWanderers {
			minions = append(minions, w)
		}
	}

	for _, s := range slashers {
		if d, p := distFromMe[s.coord]; p && d <= RangeSlashers {
			minions = append(minions, s)
		}
	}

	return minions
}

// Item : heap item
type Item struct {
	value    interface{}
	priority int
	index    int
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority < pq[j].priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push add item to heap
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

// Pop get first item by priority
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) update(value coord, priority int) {
	for _, item := range *pq {
		if item.value == value {
			item.priority = priority
			heap.Fix(pq, item.index)
			return
		}
	}
}

func logQueue(pq PriorityQueue) {
	for i, item := range pq {
		if i >= 5 {
			return
		}
		log(fmt.Sprintf("%.2d:%+v", item.priority, item.value.(coord)))
	}
}

func isTraversable(cell cell) bool {
	return cell == cellEmpty || cell == cellSpawn || cell == cellShelter
}

func getTraversableCells(grid grid) []coord {
	coords := make([]coord, 0)
	for i, line := range grid {
		for j, cell := range line {
			if isTraversable(cell) {
				coords = append(coords, coord{j, i})
			}
		}
	}
	return coords
}

func neighbors(grid grid, from coord) []coord {
	offsets := [4]coord{
		{0, -1},
		{0, 1},
		{-1, 0},
		{1, 0},
	}

	res := make([]coord, 4)
	for _, o := range offsets {
		targetCoord := coord{from.x + o.x, from.y + o.y}

		if insideGrid(grid, targetCoord) && isTraversable(grid.getCell(targetCoord)) {
			res = append(res, targetCoord)
		}
	}
	return res
}

func insideGrid(g grid, c coord) bool {
	return c.x >= 0 && c.x < len(g[0]) && c.y >= 0 && c.y < len(g)
}

func (g grid) getCell(at coord) cell {
	if !insideGrid(g, at) {
		panic(fmt.Sprintf("Coord %+v outside grid", at))
	}
	return g[at.y][at.x]
}

func checkDst(d int) {
	if d < 0 || d > 1000 {
		panic(fmt.Sprintf("distance was %d", d))
	}
}

func dijkstra(grid grid, source coord, wanderers []wanderer) (map[coord]int, map[coord]coord) {
	dist := make(map[coord]int)
	dist[source] = 0

	prev := make(map[coord]coord)

	q := make(PriorityQueue, 0)
	heap.Init(&q)

	// add all traversable cells queue
	for _, v := range getTraversableCells(grid) {
		dv, prsDv := dist[v]
		checkDst(dv)
		priority := math.MaxInt64
		if prsDv {
			priority = dv
		}
		heap.Push(&q, &Item{
			value:    v,
			priority: priority,
		})
	}

	for len(q) > 0 {
		u := heap.Pop(&q).(*Item).value.(coord)
		for _, v := range neighbors(grid, u) {
			dU, prsU := dist[u]

			if prsU {

				countWanderers := 0
				for _, w := range wanderers {
					if w.coord == v {
						countWanderers++
					}
				}

				alt := dU + 1 + countWanderers*FactWanderers
				checkDst(alt)

				dV, prsV := dist[v]

				if !prsV || alt < dV {
					dist[v] = alt
					prev[v] = u
					q.update(v, alt)
				}
			}
		}
	}

	return dist, prev
}

func main() {

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1000000), 1000000)

	var width int
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &width)

	var height int
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &height)

	currentGrid := parseGrid(scanner, width, height)
	printGrid(currentGrid)

	var sanityLossLonely, sanityLossGroup, wandererSpawnTime, wandererLifeTime int
	scanner.Scan()
	fmt.Sscan(scanner.Text(), &sanityLossLonely, &sanityLossGroup, &wandererSpawnTime, &wandererLifeTime)

	for {
		var entityCount int
		scanner.Scan()
		fmt.Sscan(scanner.Text(), &entityCount)

		explorers := make([]explorer, 0)
		wanderers := make([]wanderer, 0)
		slashers := make([]slasher, 0)
		spawningMinions := make([]spawningMinion, 0)

		for i := 0; i < entityCount; i++ {
			var entityType string
			var id, x, y, param0, param1, param2 int
			scanner.Scan()
			fmt.Sscan(scanner.Text(), &entityType, &id, &x, &y, &param0, &param1, &param2)

			switch entityType {
			case entityTypeExplorer:
				explorers = append(explorers, explorer{id, coord{x, y}, param0})
			case entityTypeWanderer:
				state := minionState(param1)
				switch state {
				case stateSpawning:
					spawningMinions = append(spawningMinions, spawningMinion{id, coord{x, y}, stateSpawning, param2, param0})
				case stateWandering:
					wanderers = append(wanderers, wanderer{id, coord{x, y}, stateWandering, param2, param0})
				default:
					panic("unrecognized state " + string(state))
				}
			case entityTypeEffectPlan:
			case entityTypeEffectLight:
			case entityTypeSlasher:
				state := minionState(param1)
				switch state {
				case stateSpawning:
					spawningMinions = append(spawningMinions, spawningMinion{id, coord{x, y}, stateSpawning, param2, param0})
				case stateWandering:
					fallthrough
				case stateStalking:
					fallthrough
				case stateRushing:
					fallthrough
				case stateStunned:
					slashers = append(slashers, slasher{id, coord{x, y}, state, param2, param0})
				default:
					panic("unrecognized state " + string(state))
				}
			case entityTypeEffectShelter:
			case entityTypeEffectYell:
			default:
				panic("unrecognized entityType " + string(entityType))
			}
		}

		log("explorers")
		for _, e := range explorers {
			log(e)
		}

		log("wanderers")
		for _, w := range wanderers {
			log(w)
		}

		log("spawning")
		for _, s := range spawningMinions {
			log(s)
		}

		log("slashers")
		for _, s := range slashers {
			log(s)
		}

		myExplorer := explorers[0]

		log("Me :")
		log(myExplorer)

		distFromMe, _ := dijkstra(currentGrid, myExplorer.coord, wanderers)
		// log("distances: ", distFromMe)
		// log("previous: ", prevFromMe)

		frighteningMinions := getFrighteningMinions(myExplorer, wanderers, slashers, distFromMe)

		if len(frighteningMinions) > 0 {
			awayMinionCoord := getAwayFromMinions(currentGrid, myExplorer, frighteningMinions, distFromMe)
			sendMove(awayMinionCoord.x, awayMinionCoord.y, "Avoiding minion")
		} else if len(explorers) > 1 {
			best := getBestExplorer(myExplorer, explorers)
			sendMove(best.x, best.y, "Following leader")
		} else {
			sendWait("Nothing to do")
		}
	}
}
