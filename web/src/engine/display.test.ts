import { describe, expect, it } from 'vitest'
import { forDisplay } from './display'

describe('forDisplay', () => {
  it('strips the emoji a foster signed off with', () => {
    expect(forDisplay('Sugar Bear is ready to be your perfect little lap warmer. 🐻💕'))
      .toBe('Sugar Bear is ready to be your perfect little lap warmer.')
  })

  it('leaves one space where an emoji sat mid sentence', () => {
    expect(forDisplay('He waits 🐕 for a lift')).toBe('He waits for a lift')
  })

  it('keeps joined and skin toned emoji from leaving pieces behind', () => {
    expect(forDisplay('my people 👨‍👩‍👧 and me 👍🏽')).toBe('my people and me')
    expect(forDisplay('flags 🇺🇸 too')).toBe('flags too')
  })

  it('changes nothing in text without emoji', () => {
    const listing = "I'm a high energy, playful dog — 49 pounds, 100% potty-pad trained (50/50 outside)."
    expect(forDisplay(listing)).toBe(listing)
  })

  it('keeps digits, accents and other alphabets', () => {
    expect(forDisplay('Canela, 4Y/1M/1W, café, Бела 🐶')).toBe('Canela, 4Y/1M/1W, café, Бела')
  })
})
