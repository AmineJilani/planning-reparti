package clock

type Stamp struct {
	//Copie dec l'horloge à un moment précis
	//On l'utilise pour envoyer les messages et comparer les priorités
	Time int
	Site string
}

type LogicalClock struct {
	// Horloge du site (interne)
	//évolue dans le temps
	Time int
	Site string
}

func NewLogicalClock(site string) *LogicalClock {
	return &LogicalClock{
		Time: 0,
		Site: site,
	}
}

func (c *LogicalClock) Tick() {
	c.Time++
}

func (c *LogicalClock) Update(received int) {
	if received >= c.Time {
		c.Time = received + 1
	} else {
		c.Time++
	}
}

func (c *LogicalClock) CurrentStamp() Stamp {
	return Stamp{
		Time: c.Time,
		Site: c.Site,
	}
}
