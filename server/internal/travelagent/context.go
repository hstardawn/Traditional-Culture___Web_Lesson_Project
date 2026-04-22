package travelagent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const defaultTimezone = "Asia/Shanghai"

type ContextBuilder struct {
	client *http.Client
	loc    *time.Location
}

func NewContextBuilder() *ContextBuilder {
	loc, err := time.LoadLocation(defaultTimezone)
	if err != nil {
		loc = time.FixedZone(defaultTimezone, 8*60*60)
	}

	return &ContextBuilder{
		client: &http.Client{Timeout: 8 * time.Second},
		loc:    loc,
	}
}

func (b *ContextBuilder) Build(ctx context.Context, req AdviceRequest) TravelContext {
	now := time.Now().In(b.loc)
	plan := b.extractPlan(req, now)
	weather := b.queryWeather(ctx, plan)
	almanac := b.queryAlmanac(plan)
	currentTime := now.Format(time.RFC3339)

	return TravelContext{
		CurrentTime: currentTime,
		CurrentDate: now.Format(time.DateOnly),
		Timezone:    b.loc.String(),
		Plan:        plan,
		Weather:     weather,
		Almanac:     almanac,
		Risks:       buildRisks(weather, almanac),
		FetchedAt:   currentTime,
	}
}

func (b *ContextBuilder) extractPlan(req AdviceRequest, now time.Time) TravelPlan {
	destination := strings.TrimSpace(req.Destination)
	if destination == "" {
		destination = extractDestination(req.Message)
	}

	travelDate := strings.TrimSpace(req.TravelDate)
	if travelDate == "" {
		travelDate = extractDate(req.Message, now, b.loc)
	}

	var needs []string
	if destination == "" {
		needs = append(needs, "目的地")
	}
	if travelDate == "" {
		needs = append(needs, "出行日期")
	}

	return TravelPlan{
		Destination:        destination,
		TravelDate:         travelDate,
		NeedsClarification: needs,
	}
}

func extractDestination(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:目的地[:：]?|去|到|前往)([\p{Han}A-Za-z0-9·.\- ]{2,24})`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(message)
		if len(matches) < 2 {
			continue
		}

		value := strings.TrimSpace(matches[1])
		if city := findKnownCity(value); city != "" {
			return city
		}
		value = strings.Trim(value, "，,。.!！？?；;")
		for _, suffix := range []string{"旅游", "旅行", "出差", "游玩", "玩", "看展", "看演出", "附近"} {
			value = strings.TrimSuffix(value, suffix)
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return findKnownCity(message)
}

func findKnownCity(text string) string {
	for _, city := range sortedKnownCityNames() {
		if strings.Contains(text, city) {
			return city
		}
	}
	return ""
}

func sortedKnownCityNames() []string {
	cities := make([]string, 0, len(knownChineseCities))
	for city := range knownChineseCities {
		cities = append(cities, city)
	}
	slices.SortFunc(cities, func(a string, b string) int {
		if lengthDiff := len([]rune(b)) - len([]rune(a)); lengthDiff != 0 {
			return lengthDiff
		}
		return strings.Compare(a, b)
	})
	return cities
}

func extractDate(message string, now time.Time, loc *time.Location) string {
	if strings.Contains(message, "后天") {
		return now.AddDate(0, 0, 2).Format(time.DateOnly)
	}
	if strings.Contains(message, "明天") {
		return now.AddDate(0, 0, 1).Format(time.DateOnly)
	}
	if strings.Contains(message, "今天") {
		return now.Format(time.DateOnly)
	}

	fullDate := regexp.MustCompile(`(20\d{2})[-/年](\d{1,2})[-/月](\d{1,2})日?`)
	if matches := fullDate.FindStringSubmatch(message); len(matches) == 4 {
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc).Format(time.DateOnly)
	}

	monthDay := regexp.MustCompile(`(\d{1,2})月(\d{1,2})日`)
	if matches := monthDay.FindStringSubmatch(message); len(matches) == 3 {
		month, _ := strconv.Atoi(matches[1])
		day, _ := strconv.Atoi(matches[2])
		date := time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, loc)
		if date.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)) {
			date = date.AddDate(1, 0, 0)
		}
		return date.Format(time.DateOnly)
	}

	isoDate := regexp.MustCompile(`20\d{2}-\d{1,2}-\d{1,2}`)
	if match := isoDate.FindString(message); match != "" {
		if parsed, err := time.ParseInLocation(time.DateOnly, match, loc); err == nil {
			return parsed.Format(time.DateOnly)
		}
	}

	return ""
}

func (b *ContextBuilder) queryWeather(ctx context.Context, plan TravelPlan) WeatherContext {
	if plan.Destination == "" {
		return WeatherContext{
			Available: false,
			Summary:   "缺少目的地，未查询天气。",
			Source:    "Open-Meteo",
		}
	}
	if plan.TravelDate == "" {
		return WeatherContext{
			Available: false,
			Location:  plan.Destination,
			Summary:   "缺少出行日期，未查询天气。",
			Source:    "Open-Meteo",
		}
	}

	location, err := b.lookupLocation(ctx, plan.Destination)
	if err != nil {
		return WeatherContext{
			Available: false,
			Location:  plan.Destination,
			Date:      plan.TravelDate,
			Summary:   err.Error(),
			Source:    "Open-Meteo",
		}
	}

	forecast, err := b.lookupForecast(ctx, location, plan.TravelDate)
	if err != nil {
		return WeatherContext{
			Available: false,
			Location:  location.Name,
			Date:      plan.TravelDate,
			Summary:   err.Error(),
			Source:    "Open-Meteo",
		}
	}

	return forecast
}

type geoLocation struct {
	Name      string
	Latitude  float64
	Longitude float64
	Timezone  string
}

var knownChineseCities = map[string]geoLocation{
	"北京": {Name: "北京 / 中国", Latitude: 39.9042, Longitude: 116.4074, Timezone: defaultTimezone},
	"上海": {Name: "上海 / 中国", Latitude: 31.2304, Longitude: 121.4737, Timezone: defaultTimezone},
	"广州": {Name: "广州 / 广东 / 中国", Latitude: 23.1291, Longitude: 113.2644, Timezone: defaultTimezone},
	"深圳": {Name: "深圳 / 广东 / 中国", Latitude: 22.5431, Longitude: 114.0579, Timezone: defaultTimezone},
	"杭州": {Name: "杭州 / 浙江 / 中国", Latitude: 30.2741, Longitude: 120.1551, Timezone: defaultTimezone},
	"南京": {Name: "南京 / 江苏 / 中国", Latitude: 32.0603, Longitude: 118.7969, Timezone: defaultTimezone},
	"苏州": {Name: "苏州 / 江苏 / 中国", Latitude: 31.2989, Longitude: 120.5853, Timezone: defaultTimezone},
	"成都": {Name: "成都 / 四川 / 中国", Latitude: 30.5728, Longitude: 104.0668, Timezone: defaultTimezone},
	"重庆": {Name: "重庆 / 中国", Latitude: 29.563, Longitude: 106.5516, Timezone: defaultTimezone},
	"西安": {Name: "西安 / 陕西 / 中国", Latitude: 34.3416, Longitude: 108.9398, Timezone: defaultTimezone},
	"武汉": {Name: "武汉 / 湖北 / 中国", Latitude: 30.5928, Longitude: 114.3055, Timezone: defaultTimezone},
	"长沙": {Name: "长沙 / 湖南 / 中国", Latitude: 28.2282, Longitude: 112.9388, Timezone: defaultTimezone},
	"厦门": {Name: "厦门 / 福建 / 中国", Latitude: 24.4798, Longitude: 118.0894, Timezone: defaultTimezone},
	"青岛": {Name: "青岛 / 山东 / 中国", Latitude: 36.0671, Longitude: 120.3826, Timezone: defaultTimezone},
	"天津": {Name: "天津 / 中国", Latitude: 39.3434, Longitude: 117.3616, Timezone: defaultTimezone},
	"大理": {Name: "大理 / 云南 / 中国", Latitude: 25.6065, Longitude: 100.2676, Timezone: defaultTimezone},
	"丽江": {Name: "丽江 / 云南 / 中国", Latitude: 26.8721, Longitude: 100.2296, Timezone: defaultTimezone},
	"桂林": {Name: "桂林 / 广西 / 中国", Latitude: 25.2342, Longitude: 110.1799, Timezone: defaultTimezone},
}

func (b *ContextBuilder) lookupLocation(ctx context.Context, name string) (geoLocation, error) {
	if location, ok := knownChineseCities[strings.TrimSpace(name)]; ok {
		return location, nil
	}

	endpoint := "https://geocoding-api.open-meteo.com/v1/search?count=1&language=zh&format=json&name=" + url.QueryEscape(name)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return geoLocation{}, err
	}

	response, err := b.client.Do(request)
	if err != nil {
		return geoLocation{}, fmt.Errorf("天气地理编码失败：%w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return geoLocation{}, fmt.Errorf("天气地理编码服务返回状态 %d", response.StatusCode)
	}

	var payload struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Timezone  string  `json:"timezone"`
			Admin1    string  `json:"admin1"`
			Country   string  `json:"country"`
		} `json:"results"`
	}

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return geoLocation{}, fmt.Errorf("天气地理编码解析失败：%w", err)
	}
	if len(payload.Results) == 0 {
		return geoLocation{}, fmt.Errorf("未找到目的地“%s”的天气坐标。", name)
	}

	result := payload.Results[0]
	labelParts := []string{result.Name}
	if result.Admin1 != "" && result.Admin1 != result.Name {
		labelParts = append(labelParts, result.Admin1)
	}
	if result.Country != "" {
		labelParts = append(labelParts, result.Country)
	}

	timezone := result.Timezone
	if timezone == "" {
		timezone = defaultTimezone
	}

	return geoLocation{
		Name:      strings.Join(labelParts, " / "),
		Latitude:  result.Latitude,
		Longitude: result.Longitude,
		Timezone:  timezone,
	}, nil
}

func (b *ContextBuilder) lookupForecast(ctx context.Context, location geoLocation, travelDate string) (WeatherContext, error) {
	values := url.Values{}
	values.Set("latitude", fmt.Sprintf("%.6f", location.Latitude))
	values.Set("longitude", fmt.Sprintf("%.6f", location.Longitude))
	values.Set("daily", "weather_code,temperature_2m_max,temperature_2m_min,precipitation_probability_max,wind_speed_10m_max")
	values.Set("timezone", location.Timezone)
	values.Set("forecast_days", "16")

	endpoint := "https://api.open-meteo.com/v1/forecast?" + values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return WeatherContext{}, err
	}

	response, err := b.client.Do(request)
	if err != nil {
		return WeatherContext{}, fmt.Errorf("天气查询失败：%w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return WeatherContext{}, fmt.Errorf("天气服务返回状态 %d", response.StatusCode)
	}

	var payload struct {
		Daily struct {
			Time                     []string  `json:"time"`
			WeatherCode              []int     `json:"weather_code"`
			Temperature2MMax         []float64 `json:"temperature_2m_max"`
			Temperature2MMin         []float64 `json:"temperature_2m_min"`
			PrecipitationProbability []float64 `json:"precipitation_probability_max"`
			WindSpeed10MMax          []float64 `json:"wind_speed_10m_max"`
		} `json:"daily"`
	}

	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return WeatherContext{}, fmt.Errorf("天气数据解析失败：%w", err)
	}

	for index, date := range payload.Daily.Time {
		if date != travelDate {
			continue
		}

		code := valueAt(payload.Daily.WeatherCode, index)
		return WeatherContext{
			Available:        true,
			Location:         location.Name,
			Date:             travelDate,
			Summary:          weatherCodeText(code),
			TemperatureMin:   valueAt(payload.Daily.Temperature2MMin, index),
			TemperatureMax:   valueAt(payload.Daily.Temperature2MMax, index),
			PrecipitationMax: valueAt(payload.Daily.PrecipitationProbability, index),
			WindSpeedMax:     valueAt(payload.Daily.WindSpeed10MMax, index),
			Source:           "Open-Meteo",
		}, nil
	}

	return WeatherContext{}, fmt.Errorf("天气预报暂不覆盖 %s，Open-Meteo 通常只提供近 16 天预报。", travelDate)
}

func (b *ContextBuilder) queryAlmanac(plan TravelPlan) AlmanacContext {
	if plan.TravelDate == "" {
		return AlmanacContext{
			Available: false,
			Note:      "缺少出行日期，未生成黄历上下文。",
			Source:    "local-almanac-rules",
		}
	}

	date, err := time.ParseInLocation(time.DateOnly, plan.TravelDate, b.loc)
	if err != nil {
		return AlmanacContext{
			Available: false,
			Date:      plan.TravelDate,
			Note:      "出行日期格式无法解析，未生成黄历上下文。",
			Source:    "local-almanac-rules",
		}
	}

	yiSets := [][]string{
		{"出行", "会友", "祈福"},
		{"纳采", "整理", "学习"},
		{"祭祀", "沐浴", "扫舍"},
		{"会友", "交易", "求医"},
		{"出行", "开市", "动身"},
		{"安床", "修整", "订盟"},
		{"祈福", "赏景", "访友"},
	}
	jiSets := [][]string{
		{"动土", "争执"},
		{"远行", "冒险涉水"},
		{"开仓", "夜行"},
		{"迁移", "急躁决策"},
		{"动土", "久留风雨"},
		{"远行", "疲劳驾驶"},
		{"诉讼", "临时改约"},
	}

	dayIndex := int(date.Unix() / 86400)
	yi := yiSets[positiveModulo(dayIndex, len(yiSets))]
	ji := jiSets[positiveModulo(dayIndex+3, len(jiSets))]

	return AlmanacContext{
		Available: true,
		Date:      plan.TravelDate,
		Yi:        yi,
		Ji:        ji,
		Note:      "黄历上下文由本地民俗规则生成，仅作传统文化参考；安全与交通判断以天气、交通和官方提示优先。",
		Source:    "local-almanac-rules",
	}
}

func buildRisks(weather WeatherContext, almanac AlmanacContext) []string {
	var risks []string

	if weather.Available {
		if weather.PrecipitationMax >= 70 {
			risks = append(risks, "降水概率较高，户外行程需要准备雨具或室内备选。")
		}
		if weather.WindSpeedMax >= 38 {
			risks = append(risks, "风力偏强，水上、山地和高处活动需要谨慎。")
		}
		if weather.TemperatureMax >= 35 {
			risks = append(risks, "最高气温偏高，建议避开午后暴晒并补水。")
		}
		if weather.TemperatureMin <= 3 {
			risks = append(risks, "最低气温较低，早晚需要保暖。")
		}
		if strings.Contains(weather.Summary, "雷") || strings.Contains(weather.Summary, "暴雨") || strings.Contains(weather.Summary, "降雪") {
			risks = append(risks, "天气现象对出行安全有明显影响，建议降低户外活动强度。")
		}
	}

	if almanac.Available && (slices.Contains(almanac.Ji, "远行") || slices.Contains(almanac.Ji, "疲劳驾驶")) {
		risks = append(risks, "黄历民俗上对远行偏保守，可作为文化提醒，但不覆盖天气和交通安全判断。")
	}

	if len(risks) == 0 {
		risks = append(risks, "未发现需要立刻调整行程的强风险；仍建议出发前复核实时天气和交通。")
	}

	return risks
}

func valueAt[T any](values []T, index int) T {
	var zero T
	if index < 0 || index >= len(values) {
		return zero
	}
	return values[index]
}

func positiveModulo(value int, base int) int {
	result := value % base
	if result < 0 {
		return result + base
	}
	return result
}

func weatherCodeText(code int) string {
	switch code {
	case 0:
		return "晴朗"
	case 1, 2:
		return "少云"
	case 3:
		return "阴天"
	case 45, 48:
		return "有雾"
	case 51, 53, 55, 56, 57:
		return "毛毛雨"
	case 61, 63, 65, 66, 67:
		return "降雨"
	case 71, 73, 75, 77:
		return "降雪"
	case 80, 81, 82:
		return "阵雨"
	case 85, 86:
		return "阵雪"
	case 95, 96, 99:
		return "雷雨"
	default:
		return "天气状况未知"
	}
}
