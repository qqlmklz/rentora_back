package services

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const nominatimSearchURL = "https://nominatim.openstreetmap.org/search"

type AddressSuggestion struct {
	Value    string `json:"value"`
	City     string `json:"city"`
	District string `json:"district"`
	Lat      string `json:"lat"`
	Lon      string `json:"lon"`
}

type AddressService struct {
	httpClient *http.Client
}

func NewAddressService() *AddressService {
	return &AddressService{
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (s *AddressService) Suggest(ctx context.Context, query string) []AddressSuggestion {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 3 {
		return []AddressSuggestion{}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	requestURL := nominatimSearchURL + "?q=" + url.QueryEscape(query) + "&format=json&addressdetails=1&limit=5"
	log.Printf("[address] nominatim query=%q", query)
	log.Printf("[address] nominatim url=%s", requestURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return []AddressSuggestion{}
	}
	req.Header.Set("User-Agent", "rentora_back")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return []AddressSuggestion{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []AddressSuggestion{}
	}
	log.Printf("RAW NOMINATIM: %s", string(body))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return []AddressSuggestion{}
	}

	var nominatimResp []struct {
		DisplayName string `json:"display_name"`
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		Name        string `json:"name"`
		Address     struct {
			City         string `json:"city"`
			Town         string `json:"town"`
			Village      string `json:"village"`
			CityDistrict string `json:"city_district"`
			Suburb       string `json:"suburb"`
		} `json:"address"`
	}
	if err := json.Unmarshal(body, &nominatimResp); err != nil {
		return []AddressSuggestion{}
	}

	out := make([]AddressSuggestion, 0, len(nominatimResp))
	for _, item := range nominatimResp {
		city := strings.TrimSpace(item.Address.City)
		if city == "" {
			city = strings.TrimSpace(item.Address.Town)
		}
		if city == "" {
			city = strings.TrimSpace(item.Address.Village)
		}
		if city == "" {
			city = strings.TrimSpace(item.Name)
		}

		district := strings.TrimSpace(item.Address.CityDistrict)
		if district == "" {
			district = strings.TrimSpace(item.Address.Suburb)
		}

		out = append(out, AddressSuggestion{
			Value:    item.DisplayName,
			City:     city,
			District: district,
			Lat:      item.Lat,
			Lon:      item.Lon,
		})
	}
	return out
}
