// Vite glob import — loads avatar URLs at build time (eager only for avatars, not SVGs)
const avatarModules = import.meta.glob('../assets/avatars/*.webp', { eager: true, import: 'default' }) as Record<string, string>

const AVATAR_URLS: string[] = Object.values(avatarModules)

/**
 * Returns a deterministic avatar URL based on a numeric ID.
 * Same ID always gets the same avatar image.
 */
export function getAvatarUrl(id: number): string {
  if (!AVATAR_URLS.length) return ''
  return AVATAR_URLS[Math.abs(id) % AVATAR_URLS.length] || ''
}

/**
 * Returns illustration URL by name (without .svg extension).
 * e.g. getIllustration('welcome-on-board')
 * Uses import.meta.url for lazy resolution — no eager glob at startup.
 */
export function getIllustration(name: string): string {
  return new URL(`../assets/illustrations/${name}.svg`, import.meta.url).href
}
