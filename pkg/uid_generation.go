package webrtc_relay

import (
	"time"
)

// var emojiList []rune = []rune("🏔🏎🚃🕤🐔🛤🚖🎿🐼🙏🏨💞🐺👽🎯🏊🍘🍕🎡🐋🍒🐜💫🏑💥⛰🎬🐝👎🚓💵📡🏤📍🍔🌐🏧👈💺🛺😳🌌🥋🐚🐄🎓🚵🔑🛖🕍💿🎚🐫🌍🌔🍓🤾💧🍌🍚💯🥘⌛🔬🛣🌊🏰🎫🌈👶🚫🚑📊💐📠👠🎤🚨🎢🐽🍞🚄🐂🍸🚗👑🦽🎹🚿⌚🎾🤿🚪🍇🐻👦🛟🥌🃏🏜🐗🚜🍫🚌🌅🪁🌳🚕🚛🚇🍵🔔🛶👏💚🤼🏄🐙😄🌄🎸🌆👙👇⛲👄🍩🩼🛳🔉💦🏃🛵🌼🏩🎅💏🌵🏠🍆🍺⭐🐣🏥💻🎮🎲👅⛱🏙📰📯🎥🍏🎊👢🐩🍍📼📺🚅⚽🚙📘🍎🚚🚀🐊🎺👧🚝🧡⛳🔕👃🛞👂🏇🍁🔫🎵🐢🖱🤺🔆🐈💔🚣🤣🪕🔦🙈🛷📸🎟🌽🚠🗼📢🍗🗜💋🎗📷🥛📫🎃💡🗿🐌🥁🎍🥝📚🍪🍟🏦💢🌬🍷😼🔩🚴🕋🎆👛🌿🚔🎽📞🏚🏵🧦🐬💭🎩⛵⛺🔧💼👻🛻🏝🍼👾🚋🐍🐸🍐🌠📽⛔🍰🪂💉📖🔍💎⛄🏘🚲📻🌀👉🎳📌🏹🔥🏀💾⛪🏍💙🏈🏕💪💒👆🍨🛫🐎🐀🚆👊😈🏟🐴💩👐🔮🐓💨🎁🔨🍬📆💣🚉👓🎀🎻🐇🏁🎪📟🎂🕌🚽🌁💃💘😠🤹👔💀🚮🪀📣🚈🏖📝💊🚒🎈🪲🥊☁🗻🏞⛷💰🎨🌇😎🚊🛼🚦🛴💬👍🛀👩🐛👌👫🎭🎷👕🌮🍃🎱🗾🕠🚢💈🍂🚧👼🔢🐧🐖🍄🎠👜🐨🛬🛹🍤🚍🤽🎄🐒🚥🐕⛩🚡🎑👰🔭⛽🎇😮🏯🍀⛑🔋⛅😌🌹😭🚘🧩🏆🍑👤🛥🐁🛰💍🛩")
// func getDailyEmoji() string {
// 	return string(emojiList[])
// }

var adjectives = [...]string{
	"Autumn", "Hidden", "Bitter", "Misty", "Silent", "Empty", "Dry", "Dark", "Summer", "Icy",
	"Delicate", "Quiet", "White", "Cool", "Spring", "Winter", "Patient", "Twilight", "Dawn",
	"Crimson", "Wispy", "Weathered", "Blue", "Billowing", "Broken", "Cold", "Damp", "Falling",
	"Frosty", "Green", "Long", "Late", "Lingering", "Bold", "Little", "Morning", "Muddy", "Old",
	"Red", "Rough", "Still", "Small", "Sparkling", "Wandering", "Withered", "Wild", "Black",
	"Young", "Holy", "Solitary", "Fragrant", "Aged", "Snowy", "Proud", "Floral", "Restless",
	"Divine", "Polished", "Ancient", "Purple", "Lively", "Nameless"}
var nouns = [...]string{
	"Waterfall", "River", "Breeze", "Moon", "Rain", "Wind", "Sea", "Morning", "Snow", "Lake",
	"Sunset", "Pine", "Shadow", "Leaf", "Dawn", "Glitter", "Forest", "Hill", "Cloud", "Meadow",
	"Sun", "Glade", "Bird", "Brook", "Butterfly", "Bush", "Dew", "Dust", "Field", "Fire",
	"Flower", "Firefly", "Feather", "Grass", "Haze", "Mountain", "Night", "Pond", "Darkness",
	"Snowflake", "Silence", "Sound", "Sky", "Shape", "Surf", "Thunder", "Violet", "Water",
	"Wildflower", "Wave", "Water", "Resonance", "Sun", "Wood", "Dream", "Cherry", "Tree", "Fog",
	"Frost", "Voice", "Paper", "Frog", "Smoke", "Star"}

// Max returns the larger of x or y.
func Max(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}

func getDailyName(offset uint64) string {
	dayOfYear := uint64(time.Time.YearDay(time.Now().UTC()))
	offset = Max(uint64(1), offset+1)
	adjective := adjectives[(offset+dayOfYear)%uint64(len(adjectives))]
	noun := nouns[(offset*dayOfYear)%uint64(len(nouns))]
	return adjective + "-" + noun
}
