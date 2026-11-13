package icon

import (
	"fmt"
	"strings"
)

// Preset is a ready made subscription the user can pick instead of typing
// everything by hand.
type Preset struct {
	// Name is the service name shown in the picker.
	Name string `json:"name"`
	// Plan is the plan description.
	Plan string `json:"plan"`
	// Category is one of the model categories.
	Category string `json:"category"`
	// Price is the monthly price in USD.
	Price float64 `json:"price"`
	// Cycle is the billing period in Vietnamese.
	Cycle string `json:"cycle"`
	// Payment is the default payment method.
	Payment string `json:"payment"`
	// IconBg is the legacy colour class used while no AI icon exists.
	IconBg string `json:"iconBg"`
	// IconText is the short label of the text badge.
	IconText string `json:"iconText"`
	// Prompt describes the artwork in generic terms. It never asks the model
	// to reproduce a brand logo, only the shape and colour of the service.
	Prompt string `json:"prompt"`
}

// Presets returns the built in catalogue of popular subscriptions.
func Presets() []Preset {
	return []Preset{
		// ------------------------------------------------------------- AI
		{
			Name: "ChatGPT Plus", Plan: "Gói cá nhân GPT-4o", Category: "AI",
			Price: 20, Cycle: "tháng", Payment: "Mastercard •••• 1092",
			IconBg: "bg-emerald-600", IconText: "GPT",
			Prompt: appIconPrompt("an AI chat assistant", "emerald green rounded square, a simple white geometric knot of two linked shapes, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Claude Pro", Plan: "Anthropic Sonnet / Opus", Category: "AI",
			Price: 20, Cycle: "tháng", Payment: "Visa •••• 4242",
			IconBg: "bg-amber-600", IconText: "CL",
			Prompt: appIconPrompt("an AI writing assistant", "warm terracotta orange rounded square, a simple abstract starburst shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Gemini Advanced", Plan: "Trợ lý AI của Google", Category: "AI",
			Price: 20, Cycle: "tháng", Payment: "Ví MoMo",
			IconBg: "bg-blue-600", IconText: "Ge",
			Prompt: appIconPrompt("a search powered AI assistant", "indigo blue rounded square, a simple four point sparkle diamond, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Canva Pro", Plan: "Thiết kế ảnh và video", Category: "AI",
			Price: 12.99, Cycle: "tháng", Payment: "Mastercard •••• 1092",
			IconBg: "bg-teal-600", IconText: "Cv",
			Prompt: appIconPrompt("a graphic design editor", "teal and cyan gradient rounded square, a simple white shape of a pen nib and a curve, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Midjourney", Plan: "Tạo ảnh AI cá nhân", Category: "AI",
			Price: 10, Cycle: "tháng", Payment: "Visa •••• 4242",
			IconBg: "bg-indigo-700", IconText: "MJ",
			Prompt: appIconPrompt("an AI image generator", "deep indigo and violet gradient rounded square, a simple white sailboat sail shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Cursor IDE", Plan: "AI Code Editor cá nhân", Category: "AI",
			Price: 20, Cycle: "tháng", Payment: "Visa •••• 4242",
			IconBg: "bg-slate-900", IconText: "Cr",
			Prompt: appIconPrompt("a code editor", "charcoal black rounded square, a simple white chevron cursor arrow, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Leonardo AI", Plan: "Tạo ảnh và game asset", Category: "AI",
			Price: 10, Cycle: "tháng", Payment: "Ví ZaloPay",
			IconBg: "bg-purple-600", IconText: "Le",
			Prompt: appIconPrompt("a generative art studio", "violet purple rounded square, a simple white geometric prism triangle, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Notion AI", Plan: "Trợ lý viết trong tài liệu", Category: "AI",
			Price: 10, Cycle: "tháng", Payment: "Visa •••• 4242",
			IconBg: "bg-neutral-800", IconText: "N",
			Prompt: appIconPrompt("a note taking app", "soft black and white rounded square, a simple outlined page corner shape, flat minimal vector style, centered, no text, no letters"),
		},

		// ---------------------------------------------------------- Media
		{
			Name: "Netflix Premium", Plan: "Gói Ultra HD 4 thiết bị", Category: "Media",
			Price: 10.50, Cycle: "tháng", Payment: "Visa •••• 8821",
			IconBg: "bg-red-700", IconText: "N",
			Prompt: appIconPrompt("a video streaming service", "deep crimson red rounded square, a simple white play triangle, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Spotify Premium", Plan: "Nghe nhạc không quảng cáo", Category: "Media",
			Price: 2.50, Cycle: "tháng", Payment: "Ví ZaloPay",
			IconBg: "bg-green-600", IconText: "Sp",
			Prompt: appIconPrompt("a music streaming service", "bright green rounded square, three simple white curved sound waves, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "YouTube Premium", Plan: "Xem video không quảng cáo", Category: "Media",
			Price: 3.20, Cycle: "tháng", Payment: "Ví MoMo",
			IconBg: "bg-red-500", IconText: "YT",
			Prompt: appIconPrompt("a video platform", "bright red rounded square, a simple white rounded play triangle, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Disney+", Plan: "Phim Disney và Marvel", Category: "Media",
			Price: 6.99, Cycle: "tháng", Payment: "Mastercard •••• 1092",
			IconBg: "bg-blue-700", IconText: "D+",
			Prompt: appIconPrompt("a family film streaming service", "deep navy blue rounded square, a simple white shooting star arc, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "FPT Play", Plan: "Xem phim và thể thao", Category: "Media",
			Price: 2.50, Cycle: "tháng", Payment: "Ví MoMo",
			IconBg: "bg-orange-600", IconText: "FP",
			Prompt: appIconPrompt("a local video streaming app", "vivid orange rounded square, a simple white rounded rectangle screen shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Apple Music", Plan: "Nghe nhạc và radio", Category: "Media",
			Price: 2.99, Cycle: "tháng", Payment: "Apple Pay (MoMo)",
			IconBg: "bg-rose-600", IconText: "AM",
			Prompt: appIconPrompt("a music streaming service", "coral red and pink gradient rounded square, a simple white music note shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Zing MP3", Plan: "Nghe nhạc và nhạc số", Category: "Media",
			Price: 1.99, Cycle: "tháng", Payment: "Ví MoMo",
			IconBg: "bg-sky-600", IconText: "Zg",
			Prompt: appIconPrompt("a music app", "bright sky blue rounded square, a simple white eighth note shape, flat minimal vector style, centered, no text, no letters"),
		},

		// ------------------------------------------------------------ Dev
		{
			Name: "GitHub Copilot", Plan: "Hỗ trợ viết code", Category: "Dev",
			Price: 10, Cycle: "tháng", Payment: "Visa •••• 4242",
			IconBg: "bg-zinc-800", IconText: "GH",
			Prompt: appIconPrompt("a developer assistant", "dark grey rounded square, a simple white rounded branch and circle shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "JetBrains All Products", Plan: "Bộ công cụ lập trình", Category: "Dev",
			Price: 20, Cycle: "tháng", Payment: "Mastercard •••• 1092",
			IconBg: "bg-blue-600", IconText: "JB",
			Prompt: appIconPrompt("a code editor suite", "royal blue rounded square, a simple white outlined square with corner brackets, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Figma Professional", Plan: "Thiết kế giao diện", Category: "Dev",
			Price: 15, Cycle: "tháng", Payment: "Visa •••• 4242",
			IconBg: "bg-purple-600", IconText: "Fg",
			Prompt: appIconPrompt("a UI design tool", "coral pink and violet rounded square, a simple white outlined overlapping circles shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Adobe Creative Cloud", Plan: "Photoshop, Illustrator", Category: "Dev",
			Price: 59.99, Cycle: "tháng", Payment: "Visa •••• 4242",
			IconBg: "bg-red-600", IconText: "Ai",
			Prompt: appIconPrompt("a creative design suite", "rich red rounded square, a simple white outlined letter A shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Docker Hub Pro", Plan: "Lưu trữ image container", Category: "Dev",
			Price: 9, Cycle: "tháng", Payment: "Mastercard •••• 1092",
			IconBg: "bg-blue-700", IconText: "Dk",
			Prompt: appIconPrompt("a container platform", "docker blue rounded square, a simple white stack of shipping containers shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Udemy Courses", Plan: "Học lập trình online", Category: "Dev",
			Price: 12.99, Cycle: "tháng", Payment: "Ví MoMo",
			IconBg: "bg-purple-600", IconText: "Ud",
			Prompt: appIconPrompt("an online learning platform", "warm purple rounded square, a simple white graduation cap shape, flat minimal vector style, centered, no text, no letters"),
		},

		// -------------------------------------------------------- Storage
		{
			Name: "Google One 2TB", Plan: "Drive + Google Photos", Category: "Storage",
			Price: 9.99, Cycle: "tháng", Payment: "Ví MoMo",
			IconBg: "bg-blue-600", IconText: "G1",
			Prompt: appIconPrompt("a cloud storage plan", "bright blue rounded square, a simple white cloud outline shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "iCloud+ 200GB", Plan: "Sao lưu iPhone & Ảnh", Category: "Storage",
			Price: 2.99, Cycle: "tháng", Payment: "Apple Pay (MoMo)",
			IconBg: "bg-sky-600", IconText: "iC",
			Prompt: appIconPrompt("a personal cloud storage plan", "pale sky blue rounded square, a simple white rounded cloud shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Dropbox Plus", Plan: "Lưu trữ và đồng bộ file", Category: "Storage",
			Price: 11.99, Cycle: "tháng", Payment: "Mastercard •••• 1092",
			IconBg: "bg-blue-600", IconText: "Db",
			Prompt: appIconPrompt("a file sync service", "bright azure blue rounded square, four simple white diamond shapes forming an open box, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "OneDrive 1TB", Plan: "Lưu trữ Microsoft 365", Category: "Storage",
			Price: 6.99, Cycle: "tháng", Payment: "Visa •••• 4242",
			IconBg: "bg-sky-600", IconText: "OD",
			Prompt: appIconPrompt("a business cloud drive", "turquoise blue rounded square, a simple white cloud shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Notion Plus", Plan: "Ghi chú & quản lý cá nhân", Category: "Storage",
			Price: 10, Cycle: "tháng", Payment: "Visa •••• 4242",
			IconBg: "bg-neutral-800", IconText: "N",
			Prompt: appIconPrompt("a personal notes app", "clean white and black rounded square, a simple outlined sheet of paper shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Evernote", Plan: "Sổ tay điện tử", Category: "Storage",
			Price: 4.99, Cycle: "tháng", Payment: "Ví MoMo",
			IconBg: "bg-green-700", IconText: "En",
			Prompt: appIconPrompt("a note taking app", "forest green rounded square, a simple white elephant head silhouette shape, flat minimal vector style, centered, no text, no letters"),
		},

		// ------------------------------------------------------- Shopping
		{
			Name: "GrabUnlimited", Plan: "Freeship đồ ăn & đi lại", Category: "Shopping",
			Price: 2, Cycle: "tháng", Payment: "Ví MoMo",
			IconBg: "bg-emerald-700", IconText: "Gr",
			Prompt: appIconPrompt("a ride hailing and delivery app", "fresh emerald green rounded square, a simple white car silhouette shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "ShopeeVIP", Plan: "Voucher freeship hàng tháng", Category: "Shopping",
			Price: 2.28, Cycle: "tháng", Payment: "ShopeePay",
			IconBg: "bg-orange-600", IconText: "Sh",
			Prompt: appIconPrompt("an online shopping marketplace", "vivid orange rounded square, a simple white shopping bag shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Lazada LPlus", Plan: "Ưu đãi và miễn phí vận chuyển", Category: "Shopping",
			Price: 2.50, Cycle: "tháng", Payment: "Ví ZaloPay",
			IconBg: "bg-blue-600", IconText: "Lz",
			Prompt: appIconPrompt("an online shopping marketplace", "sky blue rounded square, a simple white heart shape inside a bag outline, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Amazon Prime", Plan: "Giao hàng nhanh & streaming", Category: "Shopping",
			Price: 8.99, Cycle: "tháng", Payment: "Mastercard •••• 1092",
			IconBg: "bg-sky-700", IconText: "AP",
			Prompt: appIconPrompt("a fast delivery membership", "deep teal blue rounded square, a simple white curved smile arrow shape, flat minimal vector style, centered, no text, no letters"),
		},
		{
			Name: "Foodpanda Pass", Plan: "Miễn phí giao hàng ăn", Category: "Shopping",
			Price: 4.90, Cycle: "tháng", Payment: "Ví MoMo",
			IconBg: "bg-rose-600", IconText: "Fp",
			Prompt: appIconPrompt("a food delivery app", "rose pink rounded square, a simple white scooter shape, flat minimal vector style, centered, no text, no letters"),
		},
	}
}

// appIconPrompt builds a prompt for a flat application icon.
func appIconPrompt(subject, style string) string {
	return fmt.Sprintf(
		"flat minimal application icon for %s, %s, solid rounded square background, simple geometric symbol, clean vector style, high contrast, centered composition, square format, no text, no letters, no watermark",
		subject, style)
}

// BuildSubscriptionPrompt returns the icon prompt for a subscription. When the
// name matches a preset the preset prompt is used, otherwise the prompt is
// derived from the service name and its category.
func BuildSubscriptionPrompt(name, category string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "a digital service"
	}
	for _, preset := range Presets() {
		if strings.EqualFold(preset.Name, name) {
			return preset.Prompt
		}
	}

	palette := map[string]string{
		"AI":       "indigo and violet gradient",
		"Media":    "deep crimson red",
		"Dev":      "steel blue",
		"Storage":  "soft sky blue",
		"Shopping": "warm orange",
	}
	color, ok := palette[category]
	if !ok {
		color = "indigo"
	}

	return fmt.Sprintf(
		"flat minimal application icon for %s, %s rounded square, a simple white geometric symbol that suggests the purpose of the service, clean vector style, centered composition, square format, no text, no letters, no watermark",
		name, color)
}

// BuildAvatarPrompt returns the icon prompt for a Gmail account avatar.
func BuildAvatarPrompt(tag, color string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		tag = "a personal email inbox"
	}

	palette := map[string]string{
		"blue":    "azure blue",
		"purple":  "violet purple",
		"amber":   "warm amber orange",
		"emerald": "emerald green",
		"rose":    "rose pink",
	}
	tint, ok := palette[color]
	if !ok {
		tint = "azure blue"
	}

	return fmt.Sprintf(
		"flat minimal avatar app icon for %s, %s rounded square, a simple white envelope shape, clean vector style, centered composition, square format, no text, no letters, no watermark",
		tag, tint)
}

// BuildAppPrompt returns the prompt for the SubMail application icon.
func BuildAppPrompt() string {
	return "flat minimal application icon for a personal credential and subscription manager, indigo to violet gradient rounded square, a simple white combination of a padlock and a small calendar grid, clean vector style, centered composition, square format, no text, no letters, no watermark"
}
