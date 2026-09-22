import { copyFileSync } from 'node:fs'

// Ship the original copyright and complete MIT notice inside the embedded UI.
copyFileSync(new URL('../../LICENSE', import.meta.url), new URL('../public/LICENSE.txt', import.meta.url))
