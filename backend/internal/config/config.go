package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Port string
	Env  string

	// DB_DRIVER: "postgres" (default) or "sqlite"
	DBDriver   string
	DatabaseURL string
	SQLitePath  string

	// CACHE_DRIVER: "redis" (default) or "memory"
	CacheDriver string
	RedisURL    string

	JWTAccessSecret  string
	JWTRefreshSecret string

	// Provider API keys
	OpenAIAPIKey    string
	AnthropicAPIKey string
	GoogleAPIKey    string
	AlibabaAPIKey   string

	// Alipay
	AlipayAppID       string
	AlipayPrivateKey  string
	AlipayPublicKey   string
	AlipayNotifyURL   string

	// WeChat Pay
	WechatMchID      string
	WechatAppID      string
	WechatAPIV3Key   string
	WechatCertSerial string
	WechatNotifyURL  string

	// Admin seed
	AdminEmail    string
	AdminPassword string
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("DB_DRIVER", "postgres")
	viper.SetDefault("CACHE_DRIVER", "redis")
	viper.SetDefault("SQLITE_PATH", "./data/aiproxy.db")

	return &Config{
		Port: viper.GetString("PORT"),
		Env:  viper.GetString("ENV"),

		DBDriver:    viper.GetString("DB_DRIVER"),
		DatabaseURL: viper.GetString("DATABASE_URL"),
		SQLitePath:  viper.GetString("SQLITE_PATH"),

		CacheDriver: viper.GetString("CACHE_DRIVER"),
		RedisURL:    viper.GetString("REDIS_URL"),

		JWTAccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
		JWTRefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),

		OpenAIAPIKey:    viper.GetString("OPENAI_API_KEY"),
		AnthropicAPIKey: viper.GetString("ANTHROPIC_API_KEY"),
		GoogleAPIKey:    viper.GetString("GOOGLE_API_KEY"),
		AlibabaAPIKey:   viper.GetString("ALIBABA_API_KEY"),

		AlipayAppID:      viper.GetString("ALIPAY_APP_ID"),
		AlipayPrivateKey: viper.GetString("ALIPAY_PRIVATE_KEY"),
		AlipayPublicKey:  viper.GetString("ALIPAY_PUBLIC_KEY"),
		AlipayNotifyURL:  viper.GetString("ALIPAY_NOTIFY_URL"),

		WechatMchID:      viper.GetString("WECHAT_MCH_ID"),
		WechatAppID:      viper.GetString("WECHAT_APP_ID"),
		WechatAPIV3Key:   viper.GetString("WECHAT_API_V3_KEY"),
		WechatCertSerial: viper.GetString("WECHAT_CERT_SERIAL"),
		WechatNotifyURL:  viper.GetString("WECHAT_NOTIFY_URL"),

		AdminEmail:    viper.GetString("ADMIN_EMAIL"),
		AdminPassword: viper.GetString("ADMIN_PASSWORD"),
	}, nil
}
