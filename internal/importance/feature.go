package importance

import (
	"math"
	"strings"

	"github.com/masumomo/voice-brief/internal/model"
)

// Features はイベントから抽出した特徴量
type Features struct {
	// テキスト特徴量
	TextLength      float64 // 本文の長さ（正規化済み）
	TitleLength     float64 // タイトルの長さ（正規化済み）
	HasMention      float64 // @メンション有無 (0 or 1)
	HasQuestion     float64 // 疑問文か (0 or 1)
	HasExclamation  float64 // 強調表現か (0 or 1)
	HasURL          float64 // URL含有 (0 or 1)
	KeywordScore    float64 // 重要キーワードスコア（正規化済み）

	// エンゲージメント特徴量
	CommentCount      float64 // コメント数（正規化済み）
	UniqueCommenters  float64 // ユニーク投稿者数（正規化済み）
	TotalCommentLen   float64 // コメント総文字数（正規化済み）

	// カテゴリ特徴量 (one-hot)
	IsIncident float64
	IsDev      float64
	IsBiz      float64
	IsOps      float64

	// ソース特徴量 (one-hot)
	IsSlack  float64
	IsNotion float64
	IsGitHub float64

	// 時間特徴量
	HourOfDay       float64 // 時間帯（正規化: 0-1）
	IsBusinessHours float64 // 営業時間内か (0 or 1)
	IsWeekday       float64 // 平日か (0 or 1)
}

// FeatureWeights は各特徴量の重み
type FeatureWeights struct {
	TextLength      float64
	TitleLength     float64
	HasMention      float64
	HasQuestion     float64
	HasExclamation  float64
	HasURL          float64
	KeywordScore    float64
	CommentCount    float64
	UniqueCommenters float64
	TotalCommentLen float64
	IsIncident      float64
	IsDev           float64
	IsBiz           float64
	IsOps           float64
	IsSlack         float64
	IsNotion        float64
	IsGitHub        float64
	HourOfDay       float64
	IsBusinessHours float64
	IsWeekday       float64
	Bias            float64 // バイアス項
}

// DefaultWeights はデフォルトの重み設定
func DefaultWeights() *FeatureWeights {
	return &FeatureWeights{
		TextLength:       5.0,   // 長い投稿は重要な可能性
		TitleLength:      2.0,   // タイトルが長いと情報量多い
		HasMention:       15.0,  // メンションは重要
		HasQuestion:      5.0,   // 質問は対応必要
		HasExclamation:   3.0,   // 強調は注目すべき
		HasURL:           2.0,   // URL共有は情報提供
		KeywordScore:     20.0,  // キーワードマッチは重要
		CommentCount:     15.0,  // 議論が活発
		UniqueCommenters: 10.0,  // 多くの人が参加
		TotalCommentLen:  5.0,   // コメント内容が濃い
		IsIncident:       30.0,  // 障害は最重要
		IsDev:            5.0,   // 開発は中程度
		IsBiz:            3.0,   // ビジネスは中程度
		IsOps:            8.0,   // 運用は重要
		IsSlack:          0.0,   // ソースによる差はなし
		IsNotion:         2.0,   // Notionドキュメントは少し重要
		IsGitHub:         3.0,   // GitHubは開発に関連
		HourOfDay:        0.0,   // 時間帯は考慮しない
		IsBusinessHours:  2.0,   // 営業時間は少し重要
		IsWeekday:        1.0,   // 平日は少し重要
		Bias:             30.0,  // ベーススコア
	}
}

// WeightsFromConfig は設定ファイルの重みからFeatureWeightsを作成します
// nilの項目はデフォルト値が使用されます
type ConfigWeights struct {
	TextLength       *float64
	TitleLength      *float64
	HasMention       *float64
	HasQuestion      *float64
	HasExclamation   *float64
	HasURL           *float64
	KeywordScore     *float64
	CommentCount     *float64
	UniqueCommenters *float64
	TotalCommentLen  *float64
	IsIncident       *float64
	IsDev            *float64
	IsBiz            *float64
	IsOps            *float64
	IsSlack          *float64
	IsNotion         *float64
	IsGitHub         *float64
	HourOfDay        *float64
	IsBusinessHours  *float64
	IsWeekday        *float64
	Bias             *float64
}

// NewWeightsFromConfig は設定値からFeatureWeightsを作成します
func NewWeightsFromConfig(cfg *ConfigWeights) *FeatureWeights {
	if cfg == nil {
		return DefaultWeights()
	}

	defaults := DefaultWeights()

	return &FeatureWeights{
		TextLength:       valueOrDefault(cfg.TextLength, defaults.TextLength),
		TitleLength:      valueOrDefault(cfg.TitleLength, defaults.TitleLength),
		HasMention:       valueOrDefault(cfg.HasMention, defaults.HasMention),
		HasQuestion:      valueOrDefault(cfg.HasQuestion, defaults.HasQuestion),
		HasExclamation:   valueOrDefault(cfg.HasExclamation, defaults.HasExclamation),
		HasURL:           valueOrDefault(cfg.HasURL, defaults.HasURL),
		KeywordScore:     valueOrDefault(cfg.KeywordScore, defaults.KeywordScore),
		CommentCount:     valueOrDefault(cfg.CommentCount, defaults.CommentCount),
		UniqueCommenters: valueOrDefault(cfg.UniqueCommenters, defaults.UniqueCommenters),
		TotalCommentLen:  valueOrDefault(cfg.TotalCommentLen, defaults.TotalCommentLen),
		IsIncident:       valueOrDefault(cfg.IsIncident, defaults.IsIncident),
		IsDev:            valueOrDefault(cfg.IsDev, defaults.IsDev),
		IsBiz:            valueOrDefault(cfg.IsBiz, defaults.IsBiz),
		IsOps:            valueOrDefault(cfg.IsOps, defaults.IsOps),
		IsSlack:          valueOrDefault(cfg.IsSlack, defaults.IsSlack),
		IsNotion:         valueOrDefault(cfg.IsNotion, defaults.IsNotion),
		IsGitHub:         valueOrDefault(cfg.IsGitHub, defaults.IsGitHub),
		HourOfDay:        valueOrDefault(cfg.HourOfDay, defaults.HourOfDay),
		IsBusinessHours:  valueOrDefault(cfg.IsBusinessHours, defaults.IsBusinessHours),
		IsWeekday:        valueOrDefault(cfg.IsWeekday, defaults.IsWeekday),
		Bias:             valueOrDefault(cfg.Bias, defaults.Bias),
	}
}

// valueOrDefault はポインタがnilの場合デフォルト値を返します
func valueOrDefault(ptr *float64, defaultVal float64) float64 {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}

// FeatureBasedCalculator は特徴量ベースの重要度計算
type FeatureBasedCalculator struct {
	weights          *FeatureWeights
	highPriorityKeys []string
	lowPriorityKeys  []string
}

// NewFeatureBasedCalculator は新しいFeatureBasedCalculatorを作成します
func NewFeatureBasedCalculator(weights *FeatureWeights) *FeatureBasedCalculator {
	if weights == nil {
		weights = DefaultWeights()
	}
	return &FeatureBasedCalculator{
		weights: weights,
		highPriorityKeys: []string{
			"緊急", "障害", "エラー", "失敗", "ブロック", "停止",
			"urgent", "critical", "error", "failure", "blocked", "down",
			"重要", "注意", "確認", "承認", "レビュー", "リリース",
			"important", "attention", "review", "approve", "release", "deploy",
		},
		lowPriorityKeys: []string{
			"了解", "承知", "ありがとう", "thanks", "ok", "👍",
			"参加しました", "退出しました", "joined", "left",
		},
	}
}

// ExtractFeatures はイベントから特徴量を抽出します
func (c *FeatureBasedCalculator) ExtractFeatures(event *model.Event) *Features {
	f := &Features{}
	text := strings.ToLower(event.Title + " " + event.Body)

	// テキスト特徴量
	f.TextLength = sigmoid(float64(len(event.Body)) / 500.0) // 500文字で0.73程度
	f.TitleLength = sigmoid(float64(len(event.Title)) / 50.0)
	f.HasMention = boolToFloat(strings.Contains(text, "@") ||
		strings.Contains(event.Body, "<!channel>") ||
		strings.Contains(event.Body, "<!here>"))
	f.HasQuestion = boolToFloat(strings.Contains(text, "?") || strings.Contains(text, "？"))
	f.HasExclamation = boolToFloat(strings.Contains(text, "!") || strings.Contains(text, "！"))
	f.HasURL = boolToFloat(strings.Contains(text, "http://") || strings.Contains(text, "https://"))

	// キーワードスコア
	keywordScore := 0.0
	for _, kw := range c.highPriorityKeys {
		if strings.Contains(text, strings.ToLower(kw)) {
			keywordScore += 1.0
		}
	}
	for _, kw := range c.lowPriorityKeys {
		if strings.Contains(text, strings.ToLower(kw)) {
			keywordScore -= 0.5
		}
	}
	f.KeywordScore = sigmoid(keywordScore / 3.0) // 3キーワードで0.73程度

	// エンゲージメント特徴量
	f.CommentCount = sigmoid(float64(len(event.Comments)) / 5.0) // 5コメントで0.73程度
	uniqueAuthors := make(map[string]bool)
	totalLen := 0
	for _, comment := range event.Comments {
		uniqueAuthors[comment.Author] = true
		totalLen += len(comment.Text)
	}
	f.UniqueCommenters = sigmoid(float64(len(uniqueAuthors)) / 3.0)
	f.TotalCommentLen = sigmoid(float64(totalLen) / 500.0)

	// カテゴリ特徴量
	f.IsIncident = boolToFloat(event.Category == model.EventCategoryIncident)
	f.IsDev = boolToFloat(event.Category == model.EventCategoryDev)
	f.IsBiz = boolToFloat(event.Category == model.EventCategoryBiz)
	f.IsOps = boolToFloat(event.Category == model.EventCategoryOps)

	// ソース特徴量
	f.IsSlack = boolToFloat(event.Source == model.EventSourceSlack)
	f.IsNotion = boolToFloat(event.Source == model.EventSourceNotion)
	f.IsGitHub = boolToFloat(event.Source == model.EventSourceGitHub)

	// 時間特徴量
	hour := event.Timestamp.Hour()
	f.HourOfDay = float64(hour) / 24.0
	f.IsBusinessHours = boolToFloat(hour >= 9 && hour < 18)
	weekday := event.Timestamp.Weekday()
	f.IsWeekday = boolToFloat(weekday >= 1 && weekday <= 5)

	return f
}

// Calculate はイベントの重要度を計算します（0-100）
func (c *FeatureBasedCalculator) Calculate(event *model.Event) int {
	f := c.ExtractFeatures(event)
	w := c.weights

	// 線形結合でスコア計算
	score := w.Bias +
		w.TextLength*f.TextLength +
		w.TitleLength*f.TitleLength +
		w.HasMention*f.HasMention +
		w.HasQuestion*f.HasQuestion +
		w.HasExclamation*f.HasExclamation +
		w.HasURL*f.HasURL +
		w.KeywordScore*f.KeywordScore +
		w.CommentCount*f.CommentCount +
		w.UniqueCommenters*f.UniqueCommenters +
		w.TotalCommentLen*f.TotalCommentLen +
		w.IsIncident*f.IsIncident +
		w.IsDev*f.IsDev +
		w.IsBiz*f.IsBiz +
		w.IsOps*f.IsOps +
		w.IsSlack*f.IsSlack +
		w.IsNotion*f.IsNotion +
		w.IsGitHub*f.IsGitHub +
		w.HourOfDay*f.HourOfDay +
		w.IsBusinessHours*f.IsBusinessHours +
		w.IsWeekday*f.IsWeekday

	// 0-100の範囲にクリップ
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return int(score)
}

// GetFeatureVector は特徴量をスライスとして返します（デバッグ/可視化用）
func (c *FeatureBasedCalculator) GetFeatureVector(event *model.Event) []float64 {
	f := c.ExtractFeatures(event)
	return []float64{
		f.TextLength,
		f.TitleLength,
		f.HasMention,
		f.HasQuestion,
		f.HasExclamation,
		f.HasURL,
		f.KeywordScore,
		f.CommentCount,
		f.UniqueCommenters,
		f.TotalCommentLen,
		f.IsIncident,
		f.IsDev,
		f.IsBiz,
		f.IsOps,
		f.IsSlack,
		f.IsNotion,
		f.IsGitHub,
		f.HourOfDay,
		f.IsBusinessHours,
		f.IsWeekday,
	}
}

// GetFeatureNames は特徴量名のリストを返します
func GetFeatureNames() []string {
	return []string{
		"text_length",
		"title_length",
		"has_mention",
		"has_question",
		"has_exclamation",
		"has_url",
		"keyword_score",
		"comment_count",
		"unique_commenters",
		"total_comment_len",
		"is_incident",
		"is_dev",
		"is_biz",
		"is_ops",
		"is_slack",
		"is_notion",
		"is_github",
		"hour_of_day",
		"is_business_hours",
		"is_weekday",
	}
}

// sigmoid はシグモイド関数（0-1に正規化）
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// boolToFloat はboolを0.0/1.0に変換
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}
