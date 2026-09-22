export function validPassword(value) {
  return [...value].length >= 8 && new TextEncoder().encode(value).length <= 72
}
