package main

import (
	"math/rand"
)

const maxAdsToServe = 2

// Service implements the business logic for AdService.
type Service struct {
	adsMap map[string][]Ad
}

// NewService creates a new AdService with pre-populated data.
func NewService() *Service {
	return &Service{
		adsMap: createAdsMap(),
	}
}

// GetAdsByCategory returns ads matching the given categories.
// If no categories are provided or no matches found, it falls back to random ads.
func (s *Service) GetAdsByCategory(categories []string) []Ad {
	var allAds []Ad

	if len(categories) > 0 {
		for _, category := range categories {
			if ads, ok := s.adsMap[category]; ok {
				allAds = append(allAds, ads...)
			}
		}
	} else {
		return s.GetRandomAds()
	}

	if len(allAds) == 0 {
		return s.GetRandomAds()
	}

	return allAds
}

// GetRandomAds returns a random selection of ads.
func (s *Service) GetRandomAds() []Ad {
	var allAds []Ad
	for _, ads := range s.adsMap {
		allAds = append(allAds, ads...)
	}

	if len(allAds) == 0 {
		return []Ad{}
	}

	// Shuffle and pick
	shuffled := make([]Ad, len(allAds))
	copy(shuffled, allAds)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	limit := maxAdsToServe
	if len(shuffled) < limit {
		limit = len(shuffled)
	}

	return shuffled[:limit]
}

func createAdsMap() map[string][]Ad {
	hairdryer := Ad{RedirectURL: "/product/2ZYFJ3GM2N", Text: "Hairdryer for sale. 50% off."}
	tankTop := Ad{RedirectURL: "/product/66VCHSJNUP", Text: "Tank top for sale. 20% off."}
	candleHolder := Ad{RedirectURL: "/product/0PUK6V6EV0", Text: "Candle holder for sale. 30% off."}
	bambooGlassJar := Ad{RedirectURL: "/product/9SIQT8TOJO", Text: "Bamboo glass jar for sale. 10% off."}
	watch := Ad{RedirectURL: "/product/1YMWWN1N4O", Text: "Watch for sale. Buy one, get second kit for free"}
	mug := Ad{RedirectURL: "/product/6E92ZMYYFZ", Text: "Mug for sale. Buy two, get third one for free"}
	loafers := Ad{RedirectURL: "/product/L9ECAV7KIM", Text: "Loafers for sale. Buy one, get second one for free"}

	return map[string][]Ad{
		"clothing":    {tankTop},
		"accessories": {watch},
		"footwear":    {loafers},
		"hair":        {hairdryer},
		"decor":       {candleHolder},
		"kitchen":     {bambooGlassJar, mug},
	}
}
