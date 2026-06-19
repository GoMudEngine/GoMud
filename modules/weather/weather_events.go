package weather

// WeatherSeasonChanged is queued on the engine event bus when a zone's
// resolved season flips on a calendar boundary. From/To are season names
// within the same Track. Other modules (crops, forage, ...) can listen:
//
//	events.RegisterListener(weather.WeatherSeasonChanged{}, handler)
type WeatherSeasonChanged struct {
	Zone  string
	Track string
	From  string
	To    string
}

func (WeatherSeasonChanged) Type() string { return `WeatherSeasonChanged` }
