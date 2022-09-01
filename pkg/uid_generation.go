package webrtc_relay

// var emojiList []rune = []rune("🏔🏎🚃🕤🐔🛤🚖🎿🐼🙏🏨💞🐺👽🎯🏊🍘🍕🎡🐋🍒🐜💫🏑💥⛰🎬🐝👎🚓💵📡🏤📍🍔🌐🏧👈💺🛺😳🌌🥋🐚🐄🎓🚵🔑🛖🕍💿🎚🐫🌍🌔🍓🤾💧🍌🍚💯🥘⌛🔬🛣🌊🏰🎫🌈👶🚫🚑📊💐📠👠🎤🚨🎢🐽🍞🚄🐂🍸🚗👑🦽🎹🚿⌚🎾🤿🚪🍇🐻👦🛟🥌🃏🏜🐗🚜🍫🚌🌅🪁🌳🚕🚛🚇🍵🔔🛶👏💚🤼🏄🐙😄🌄🎸🌆👙👇⛲👄🍩🩼🛳🔉💦🏃🛵🌼🏩🎅💏🌵🏠🍆🍺⭐🐣🏥💻🎮🎲👅⛱🏙📰📯🎥🍏🎊👢🐩🍍📼📺🚅⚽🚙📘🍎🚚🚀🐊🎺👧🚝🧡⛳🔕👃🛞👂🏇🍁🔫🎵🐢🖱🤺🔆🐈💔🚣🤣🪕🔦🙈🛷📸🎟🌽🚠🗼📢🍗🗜💋🎗📷🥛📫🎃💡🗿🐌🥁🎍🥝📚🍪🍟🏦💢🌬🍷😼🔩🚴🕋🎆👛🌿🚔🎽📞🏚🏵🧦🐬💭🎩⛵⛺🔧💼👻🛻🏝🍼👾🚋🐍🐸🍐🌠📽⛔🍰🪂💉📖🔍💎⛄🏘🚲📻🌀👉🎳📌🏹🔥🏀💾⛪🏍💙🏈🏕💪💒👆🍨🛫🐎🐀🚆👊😈🏟🐴💩👐🔮🐓💨🎁🔨🍬📆💣🚉👓🎀🎻🐇🏁🎪📟🎂🕌🚽🌁💃💘😠🤹👔💀🚮🪀📣🚈🏖📝💊🚒🎈🪲🥊☁🗻🏞⛷💰🎨🌇😎🚊🛼🚦🛴💬👍🛀👩🐛👌👫🎭🎷👕🌮🍃🎱🗾🕠🚢💈🍂🚧👼🔢🐧🐖🍄🎠👜🐨🛬🛹🍤🚍🤽🎄🐒🚥🐕⛩🚡🎑👰🔭⛽🎇😮🏯🍀⛑🔋⛅😌🌹😭🚘🧩🏆🍑👤🛥🐁🛰💍🛩")
// func getDailyEmoji() string {
// dayOfYear := uint32(time.Time.YearDay(time.Now().UTC()))
// 	return string(emojiList[])
// }

var adjectives = [...]string{"Ancient", "Dawn", "Small", "Broken", "Red", "Cold", "Wild", "Divine", "Empty", "Patient", "Holy", "Long", "Wispy", "White", "Delicate", "Bold", "Billowing", "Blue", "Crimson", "Aged", "Misty", "Snowy", "Withered", "Little", "Frosty", "Weathered", "Nameless", "Fragrant", "Lively", "Quiet", "Purple", "Proud", "Dry", "Bitter", "Dark", "Icy", "Twilight", "Wandering", "Solitary", "Morning", "Lingering", "Still", "Late", "Sparkling", "Restless", "Winter", "Silent", "Floral", "Young", "Green", "Cool", "Autumn", "Falling", "Spring", "Summer", "Polished", "Hidden", "Damp", "Muddy", "Black", "Old", "Rough"}
var nouns = [...]string{"Pond", "Snow", "Glade", "Hill", "Voice", "River", "Sun", "Dawn", "Forest", "Frog", "Grass", "Shadow", "Dust", "Water", "Meadow", "Moon", "Thunder", "Sun", "Wildflower", "Snowflake", "Silence", "Haze", "Shape", "Pine", "Waterfall", "Sound", "Wood", "Tree", "Night", "Flower", "Dream", "Cherry", "Resonance", "Firefly", "Bush", "Star", "Darkness", "Lake", "Frost", "Paper", "Surf", "Fog", "Brook", "Mountain", "Field", "Bird", "Leaf", "Sea", "Water", "Sky", "Smoke", "Sunset", "Glitter", "Dew", "Butterfly", "Wind", "Fire", "Rain", "Morning", "Feather", "Cloud", "Breeze", "Violet", "Wave"}

// Max returns the larger of x or y.
func Max(x, y uint32) uint32 {
	if x < y {
		return y
	}
	return x
}

func getUniqueName(num uint32, offset uint32) string {
	adjectivesLen := uint32(len(adjectives))
	nounsLen := uint32(len(nouns))
	maxLen := Max(adjectivesLen, nounsLen)
	// prevent int overflows:
	offsetI := Max(uint32(1), (offset+1)%maxLen)
	numI := Max(uint32(1), (num+1)%maxLen)
	// get words
	adjective := adjectives[(numI+offsetI)%adjectivesLen]
	noun := nouns[(numI*offsetI)%nounsLen]
	return adjective + "-" + noun
}

/*
// getUniqueName() testcases:
var maxUINT32 uint32 = 4294967295
var maxUINT32PlusOne uint32 = maxUINT32
maxUINT32PlusOne += 1
var maxUINT32Less uint32 = 4294967292
var zero uint32 = 0

println("Max UINT32 value")
println(getUniqueName(maxUINT32, maxUINT32))
println("min Max UINT32 value + 1")
println(getUniqueName(maxUINT32Less, maxUINT32PlusOne))
println("Zero and Max UINT32 value")
println(getUniqueName(zero, maxUINT32PlusOne))
println("Zero and  Max UINT32 value + 1")
println(getUniqueName(maxUINT32PlusOne, zero))
println("Zero and LT Max UINT32 value")
println(getUniqueName(maxUINT32Less, maxUINT32Less))
*/
