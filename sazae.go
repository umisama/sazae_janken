package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"
)

const (
	HISTORY_LENGTH    = 4
	CROSSOVER_LENGTH  = 10
	CROSSOVER_PERMILL = 500
	MUTATION_PERMILLI = 30
	MUTATION_LENGTH   = 10

	TOURNAMENT_TOTAL = 100
	TOURNAMENT_SIZE  = 35

	MAX_GENERATION = 30

	POINT_WIN = 1
	POINT_EVEN = 0
	POINT_LOSE = 0
)

type Hand int

const (
	Hand_Gu = Hand(iota)
	Hand_Choki
	Hand_Pa
)

func (h Hand) String() string {
	switch h {
	case Hand_Gu:
		return "G"
	case Hand_Choki:
		return "C"
	case Hand_Pa:
		return "P"
	}
	return "panic"
}

func (h Hand) Point(t Hand) int {
	point := 0
	switch h {
	case Hand_Gu:
		switch t {
		case Hand_Gu:
			point = POINT_EVEN
		case Hand_Choki:
			point = POINT_WIN
		case Hand_Pa:
			point = POINT_LOSE
		}
	case Hand_Choki:
		switch t {
		case Hand_Gu:
			point = POINT_LOSE
		case Hand_Choki:
			point = POINT_EVEN
		case Hand_Pa:
			point = POINT_WIN
		}
	case Hand_Pa:
		switch t {
		case Hand_Gu:
			point = POINT_WIN
		case Hand_Choki:
			point = POINT_LOSE
		case Hand_Pa:
			point = POINT_EVEN
		}
	}
	return point
}

func GetRandHand() Hand {
	return Hand(rand.Int() % 3)
}

type Gene struct {
	decision []Hand
}

func NewGene() *Gene {
	decision := make([]Hand, decisionlength())
	for i, _ := range decision {
		decision[i] = GetRandHand()
	}

	return &Gene{
		decision: decision,
	}
}

func (g *Gene) Score(hands []Hand) int {
	score := 0
	for i := HISTORY_LENGTH + 1; i < len(hands)-1; i++ {
		score += g.Hand(hands[i-HISTORY_LENGTH-1 : i-1]).Point(hands[i])
	}
	return score
}

func (g *Gene) Hand(history []Hand) Hand {
	if len(history) != HISTORY_LENGTH {
		panic("invalid history")
	}

	var index int
	for _, v := range history {
		index += index*3 + int(v)
	}
	return g.decision[index]
}
/*
func (g *Gene) CrossOver(partner *Gene) *Gene {
	new_decision := make([]Hand, decisionlength())
	copy(new_decision, g.decision)

	// crossover
	ptr := rand.Int() % (decisionlength() - CROSSOVER_LENGTH)
	for i := 0; i < CROSSOVER_LENGTH; i++ {
		new_decision[i+ptr] = partner.decision[i+ptr]
	}

	return &Gene{
		decision: new_decision,
	}
}
*/

func (g *Gene) CrossOver(partner *Gene) *Gene {
	new_decision := make([]Hand,decisionlength())
	copy(new_decision, g.decision)

	for i, _ := range new_decision {
		if rand.Int()%1000 < CROSSOVER_PERMILL {
			new_decision[i] = partner.decision[i]
		}
	}

	return &Gene{
		decision: new_decision,
	}
}

func (g *Gene) Mutation() {
	if r := rand.Int() % 1000; r < MUTATION_PERMILLI {
		g.doMutation()
	}
	return
}

func (g *Gene) doMutation() *Gene {
	ptr := rand.Int() % (decisionlength() - MUTATION_LENGTH)
	for i := 0; i < MUTATION_LENGTH; i++ {
		g.decision[i+ptr] = Hand(rand.Int() % 3)
	}
	return g
}

func CreateHistory(filename string, start, end time.Time) []Hand {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	buf := make([]map[string]interface{}, 0)
	err = json.NewDecoder(f).Decode(&buf)
	if err != nil {
		panic(err)
	}

	hands := make([]Hand, 0)
	for _, v := range buf {
		if t, _ := time.Parse(time.RFC3339, v["when"].(string)); t.Before(start) {
			continue
		}
		if t, _ := time.Parse(time.RFC3339, v["when"].(string)); t.After(end) {
			continue
		}
		if err != nil {
			panic(err)
		}
		hands = append(hands, Hand(int(v["hand"].(float64))))
	}

	return hands
}

type ResultList []Resulter
type Resulter struct {
	score  int
	number int
}

func (rl ResultList) Len() int {
	return len(rl)
}

func (rl ResultList) Swap(i, j int) {
	rl[i], rl[j] = rl[j], rl[i]
}

func (rl ResultList) Less(i, j int) bool {
	return rl[i].score > rl[j].score
}

func main() {
	// initialize history
	hist := CreateHistory("sazae.json", mustParseTime("2012-08-01T00:00:00Z"), mustParseTime("2014-08-01T00:00:00Z"))
	fmt.Println(len(hist))

	// initialize
	genes := make([]*Gene, TOURNAMENT_TOTAL)
	for i, _ := range genes {
		genes[i] = NewGene()
	}

	lasttop := 0
	for gen := 0; gen < MAX_GENERATION; gen++ {
		// Tournament
		results := make(ResultList, 0)
		for i, gene := range genes {
			results = append(results, Resulter{
				number: i,
				score:  gene.Score(hist),
			})
		}
		sort.Stable(results)

		fmt.Printf("gene:%3d / topscore:%3d / next:%s\n", gen, results[0].score, genes[results[0].number].Hand(hist[len(hist)-HISTORY_LENGTH:] ))
		lasttop = results[0].score

		// Get winner
		winner := make([]*Gene, TOURNAMENT_SIZE)
		for i:=0; i<TOURNAMENT_SIZE; i++ {
			winner[i] = genes[results[i].number]
		}

		next_genes := make([]*Gene, TOURNAMENT_TOTAL)
		for i:=0; i<TOURNAMENT_TOTAL-TOURNAMENT_SIZE; i++ {
			j := rand.Int()%(TOURNAMENT_SIZE-1)
			next_genes[i] = winner[j].CrossOver(winner[j+1])
		}
		for i:=0; i<TOURNAMENT_SIZE; i++ {
			next_genes[i+TOURNAMENT_TOTAL-TOURNAMENT_SIZE] = winner[i]
		}

		genes = next_genes
	}

	fmt.Println(float64(lasttop)/float64(len(hist)))
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func decisionlength() int {
	f64HistLen := float64(HISTORY_LENGTH)
	return int(math.Pow(f64HistLen, f64HistLen*2))
}

func mustParseTime(str string) time.Time {
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		panic(err)
	}
	return t
}
