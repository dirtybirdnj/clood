// Package flyingcats provides static text generation for structured prompt scaffolding.
//
// Named for the Flying Cats of Delaware - legendary felines who nest in radio towers
// and antenna structures along the highway. Their stories, first told on long car trips,
// featured the protagonist Pooparoo (a Toonces-like cat of questionable driving ability).
//
// Flying cats sit atop the catfight architecture, adding structural randomness to prompts
// without requiring LLM inference. This is the special sauce that helps the system
// reason about what it's doing before the models ever see the prompt.
package flyingcats

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Generator creates structured nonsense from patterns
type Generator struct {
	rng *rand.Rand
}

// NewGenerator creates a new flying cats generator
func NewGenerator() *Generator {
	return &Generator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewSeededGenerator creates a generator with a specific seed for reproducibility
func NewSeededGenerator(seed int64) *Generator {
	return &Generator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// Generate produces structured nonsense based on the query
func (g *Generator) Generate(query string) string {
	query = strings.ToLower(query)

	var parts []string

	// Parse for structural elements
	if count := g.extractCount(query, "actor"); count > 0 {
		parts = append(parts, g.generateActors(count))
	}
	if count := g.extractCount(query, "person"); count > 0 {
		parts = append(parts, g.generateActors(count))
	}
	if count := g.extractCount(query, "character"); count > 0 {
		parts = append(parts, g.generateActors(count))
	}

	if strings.Contains(query, "meal") || strings.Contains(query, "food") || strings.Contains(query, "eat") {
		parts = append(parts, g.generateMeal())
	}

	if strings.Contains(query, "restaurant") || strings.Contains(query, "diner") || strings.Contains(query, "cafe") {
		parts = append(parts, g.generateRestaurant())
	}

	if strings.Contains(query, "street") || strings.Contains(query, "road") || strings.Contains(query, "address") {
		parts = append(parts, g.generateStreet())
	}

	if strings.Contains(query, "city") || strings.Contains(query, "town") || strings.Contains(query, "place") {
		parts = append(parts, g.generateCity())
	}

	if strings.Contains(query, "antenna") || strings.Contains(query, "tower") || strings.Contains(query, "radio") {
		parts = append(parts, g.generateAntennaTower())
	}

	if strings.Contains(query, "cat") || strings.Contains(query, "feline") {
		parts = append(parts, g.generateFlyingCat())
	}

	if strings.Contains(query, "option") || strings.Contains(query, "choice") || strings.Contains(query, "alternative") {
		count := g.extractCount(query, "option")
		if count == 0 {
			count = 3 // default to 3 options
		}
		parts = append(parts, g.generateOptions(count))
	}

	if strings.Contains(query, "reason") || strings.Contains(query, "why") || strings.Contains(query, "because") {
		parts = append(parts, g.generateReason())
	}

	if strings.Contains(query, "time") || strings.Contains(query, "when") || strings.Contains(query, "hour") {
		parts = append(parts, g.generateTime())
	}

	if strings.Contains(query, "name") {
		parts = append(parts, g.generateName())
	}

	// If nothing matched, generate something creative based on query words
	if len(parts) == 0 {
		parts = append(parts, g.generateFreeform(query))
	}

	return strings.Join(parts, "\n\n")
}

// extractCount finds patterns like "three actors" or "5 people"
func (g *Generator) extractCount(query string, noun string) int {
	// Word numbers
	wordNums := map[string]int{
		"one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
		"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
		"a": 1, "an": 1, "some": 3, "several": 4, "many": 5, "few": 3,
	}

	// Look for "N noun" pattern
	patterns := []string{
		`(\d+)\s+` + noun,
		`(\w+)\s+` + noun,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			if n, err := strconv.Atoi(matches[1]); err == nil {
				return n
			}
			if n, ok := wordNums[matches[1]]; ok {
				return n
			}
		}
	}

	// Check for plural without number (assume 1)
	if strings.Contains(query, noun+"s") {
		return 2
	}
	if strings.Contains(query, noun) {
		return 1
	}

	return 0
}

// Word banks - the nesting grounds of the flying cats

var actors = []string{
	"Danny DeVito", "Burt Reynolds", "Dolly Parton", "Christopher Walken",
	"Nicolas Cage", "Jeff Goldblum", "Tilda Swinton", "Bill Murray",
	"Cate Blanchett", "Keanu Reeves", "Frances McDormand", "Willem Dafoe",
	"Sigourney Weaver", "Gary Oldman", "Meryl Streep", "John Malkovich",
	"Toonces the Driving Cat", "Pooparoo of Delaware", "The Third Antenna Cat",
	"Steve Buscemi", "Aubrey Plaza", "Oscar Isaac", "Florence Pugh",
}

var meals = []string{
	"spaghetti with too much parmesan", "breakfast for dinner",
	"a single hard-boiled egg", "seventeen varieties of cheese",
	"pho with extra sriracha", "a questionable casserole",
	"nachos that have seen better days", "pad thai from the good place",
	"an extremely tall sandwich", "soup that's more bread than liquid",
	"fish and chips wrapped in newspaper", "a burrito the size of a forearm",
	"waffles with suspicious toppings", "ramen at 2am",
	"grilled cheese with pickle juice", "the antenna tower special",
}

var restaurants = []string{
	"The Screaming Radiator", "Pooparoo's Diner", "Delaware Tower Cafe",
	"Cats & Carburetors", "The Rusty Antenna", "Highway 95 Eatery",
	"Toonces Memorial Grill", "The Flying Feline Bistro",
	"Last Chance Before the Tunnel", "Muffler & Muffins",
	"The Existential Diner", "Breakfast at Tiffany's Automotive",
	"Chez Broadcast Tower", "The Satellite Dish Buffet",
	"Radio Wave Ramen", "The Third Exit Cantina",
}

var streets = []string{
	"Antenna Avenue", "Broadcast Lane", "Pooparoo Way",
	"Delaware Tower Road", "Highway Overpass Circle",
	"Radio Mast Boulevard", "Toonces Memorial Drive",
	"Flying Cat Lane", "Transmission Street", "Signal Path",
	"Frequency Court", "Wavelength Way", "The Old Toll Road",
	"Interstate Adjacent Drive", "Exit Ramp Place",
	"Service Road 17B", "Parallel to the Highway Street",
}

var cities = []string{
	"North Delaware Junction", "Antenna Heights", "Broadcast City",
	"Pooparoo Falls", "Tower Grove", "Signal Station",
	"Frequency Town", "Wavelength Valley", "The Rest Stop",
	"Exit 47 Township", "Radio Tower Vista", "Toll Plaza Point",
	"Highway Adjacent Hamlet", "Flying Cat Crossing",
	"Lesser Wilmington", "East of the Big Tower",
}

var antennaTowers = []string{
	"the Old Red Tower by exit 12", "that really tall one with the blinking light",
	"the three-legged tower where the cats gather", "the AM/FM hybrid spire",
	"the decommissioned broadcast tower", "Pooparoo's favorite roost",
	"the tower that hums when it rains", "the rusty lattice near the diner",
	"the one you can see from three states", "the emergency broadcast tower",
	"the cell tower disguised as a tree (poorly)", "the historic radio mast",
	"the tower with the cat-sized maintenance platform",
}

var flyingCatNames = []string{
	"Pooparoo", "Toonces Jr.", "Whiskers McBroadcast", "Sir Flaps-a-Lot",
	"Antenna Annie", "Radio Rex", "Frequency Fiona", "Wavelength Walter",
	"The Delaware Drifter", "Static Steve", "Tower Tom",
	"Broadcast Betty", "Signal Sally", "Mast Master Mike",
	"The One Who Watches from the Blinking Light",
}

var flyingCatDescriptions = []string{
	"known for their questionable navigation skills",
	"last seen circling the AM transmitter",
	"rumored to control the weather with their whiskers",
	"banned from three radio stations",
	"fluent in Morse code",
	"surprisingly good at parallel parking (unlike Toonces)",
	"often mistaken for a large pigeon at dawn",
	"responsible for that weird static last Tuesday",
	"has lived in the tower since the Carter administration",
	"claims to have invented the emergency broadcast tone",
}

var adjectives = []string{
	"peculiar", "magnificent", "slightly damp", "inexplicable",
	"radio-frequency-adjacent", "tower-dwelling", "highway-haunting",
	"mysteriously broadcast", "partially visible", "somewhat legendary",
	"antenna-blessed", "frequency-touched", "signal-strong",
	"Delaware-certified", "toll-road-approved", "exit-ramp-famous",
}

var reasons = []string{
	"because the antenna demanded it",
	"for reasons that remain unclear to this day",
	"as foretold by the radio static",
	"due to a misunderstanding about toll booth protocol",
	"because Pooparoo said so",
	"in accordance with Flying Cat Law",
	"to maintain the delicate balance of broadcast frequencies",
	"as is tradition on the Delaware highway",
	"because the blinking red light willed it",
	"for the same reason cats do anything: chaos",
}

var timeExpressions = []string{
	"during the 3am broadcast lull", "at the stroke of rush hour",
	"when the antenna casts its longest shadow",
	"just before the toll booth opens", "at exactly sunset",
	"during the emergency broadcast test", "in the witching hour of static",
	"when traffic slows to a crawl", "at the moment of perfect reception",
	"during the mysterious dead air at 4:47am",
}

// Generation methods

func (g *Generator) generateActors(count int) string {
	selected := g.pickN(actors, count)
	if count == 1 {
		return fmt.Sprintf("Actor: %s", selected[0])
	}
	var lines []string
	for i, actor := range selected {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, actor))
	}
	return fmt.Sprintf("Actors (%d):\n%s", count, strings.Join(lines, "\n"))
}

func (g *Generator) generateMeal() string {
	meal := meals[g.rng.Intn(len(meals))]
	adj := adjectives[g.rng.Intn(len(adjectives))]
	return fmt.Sprintf("The meal: %s %s", adj, meal)
}

func (g *Generator) generateRestaurant() string {
	rest := restaurants[g.rng.Intn(len(restaurants))]
	city := cities[g.rng.Intn(len(cities))]
	return fmt.Sprintf("Restaurant: %s, located in %s", rest, city)
}

func (g *Generator) generateStreet() string {
	street := streets[g.rng.Intn(len(streets))]
	num := g.rng.Intn(9999) + 1
	return fmt.Sprintf("Street: %d %s", num, street)
}

func (g *Generator) generateCity() string {
	city := cities[g.rng.Intn(len(cities))]
	return fmt.Sprintf("City: %s", city)
}

func (g *Generator) generateAntennaTower() string {
	tower := antennaTowers[g.rng.Intn(len(antennaTowers))]
	return fmt.Sprintf("Antenna Tower: %s", tower)
}

func (g *Generator) generateFlyingCat() string {
	name := flyingCatNames[g.rng.Intn(len(flyingCatNames))]
	desc := flyingCatDescriptions[g.rng.Intn(len(flyingCatDescriptions))]
	tower := antennaTowers[g.rng.Intn(len(antennaTowers))]
	return fmt.Sprintf("Flying Cat: %s, %s.\nCurrently residing in %s.", name, desc, tower)
}

func (g *Generator) generateOptions(count int) string {
	// Generate diverse options by mixing elements
	var options []string
	for i := 0; i < count; i++ {
		option := g.generateSingleOption()
		options = append(options, fmt.Sprintf("Option %d: %s", i+1, option))
	}
	return strings.Join(options, "\n")
}

func (g *Generator) generateSingleOption() string {
	templates := []string{
		"Visit %s in %s for %s",
		"Have %s deliver %s to %s",
		"Meet at %s, then proceed to %s for %s",
		"Take %s to see %s at %s",
		"Arrange for %s to prepare %s at %s",
	}

	template := templates[g.rng.Intn(len(templates))]

	// Fill in template based on format specifiers
	count := strings.Count(template, "%s")
	var args []interface{}
	for i := 0; i < count; i++ {
		switch i % 3 {
		case 0:
			args = append(args, g.pick(restaurants, actors, antennaTowers))
		case 1:
			args = append(args, g.pick(cities, streets, antennaTowers))
		case 2:
			args = append(args, g.pick(meals, flyingCatNames, reasons))
		}
	}

	return fmt.Sprintf(template, args...)
}

func (g *Generator) generateReason() string {
	return reasons[g.rng.Intn(len(reasons))]
}

func (g *Generator) generateTime() string {
	return timeExpressions[g.rng.Intn(len(timeExpressions))]
}

func (g *Generator) generateName() string {
	// Mix flying cat names with actors for variety
	if g.rng.Float32() < 0.4 {
		return flyingCatNames[g.rng.Intn(len(flyingCatNames))]
	}
	return actors[g.rng.Intn(len(actors))]
}

func (g *Generator) generateFreeform(query string) string {
	// When we can't parse the structure, generate something evocative
	templates := []string{
		"In the shadow of %s, %s once remarked: '%s'. This was %s.",
		"According to the archives of %s, the answer lies %s, where %s %s.",
		"The flying cats of %s maintain that %s, especially %s.",
		"Between %s and %s, you'll find %s, %s.",
		"As %s told %s at %s: the truth is %s.",
	}

	template := templates[g.rng.Intn(len(templates))]
	count := strings.Count(template, "%s")
	var args []interface{}
	allBanks := [][]string{actors, cities, antennaTowers, meals, restaurants, streets, reasons, adjectives}
	for i := 0; i < count; i++ {
		bank := allBanks[g.rng.Intn(len(allBanks))]
		args = append(args, bank[g.rng.Intn(len(bank))])
	}

	return fmt.Sprintf(template, args...)
}

// Helper methods

func (g *Generator) pickN(bank []string, n int) []string {
	if n >= len(bank) {
		n = len(bank)
	}
	// Fisher-Yates shuffle copy
	shuffled := make([]string, len(bank))
	copy(shuffled, bank)
	for i := len(shuffled) - 1; i > 0; i-- {
		j := g.rng.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled[:n]
}

func (g *Generator) pick(banks ...[]string) string {
	bank := banks[g.rng.Intn(len(banks))]
	return bank[g.rng.Intn(len(bank))]
}

// Chaos mode - pure stream of consciousness
func (g *Generator) Chaos() string {
	lines := []string{
		g.generateFlyingCat(),
		"",
		fmt.Sprintf("Witnessed %s, %s.", timeExpressions[g.rng.Intn(len(timeExpressions))], reasons[g.rng.Intn(len(reasons))]),
		"",
		g.generateOptions(g.rng.Intn(3) + 2),
		"",
		g.generateFreeform("chaos"),
	}
	return strings.Join(lines, "\n")
}

// Antenna mode - Delaware highway tower themed
func (g *Generator) Antenna() string {
	lines := []string{
		"=== TRANSMISSION FROM THE TOWERS OF DELAWARE ===",
		"",
		g.generateAntennaTower(),
		"",
		g.generateFlyingCat(),
		"",
		fmt.Sprintf("Signal Status: %s", adjectives[g.rng.Intn(len(adjectives))]),
		fmt.Sprintf("Broadcast Time: %s", timeExpressions[g.rng.Intn(len(timeExpressions))]),
		fmt.Sprintf("Tower Wisdom: %s", reasons[g.rng.Intn(len(reasons))]),
		"",
		"The cats remember. The towers know.",
	}
	return strings.Join(lines, "\n")
}
