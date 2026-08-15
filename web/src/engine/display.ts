// Display layer for text that came from a real listing. The record keeps
// the foster's words exactly as written, the screen keeps the house style.
// Never call this before storing, only on the way to the player.

// Pictographs, skin tones, flags, the zero width joiner and the emoji
// variation selector. Digits and punctuation are deliberately left alone.
const emoji = /[\p{Extended_Pictographic}\p{Emoji_Modifier}\p{Regional_Indicator}\u200D\uFE0F]/gu

// forDisplay renders a quoted fact or any description derived text.
// Emoji come out, the words and their order stay untouched.
export function forDisplay(text: string): string {
  return text
    .replace(emoji, ' ')
    .replace(/[ \t]+/g, ' ')
    .replace(/ ([,.!?;:])/g, '$1')
    .trim()
}
