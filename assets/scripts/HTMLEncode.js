/**!
 * @name          HTML Encode
 * @description   Encodes HTML entities in your text
 * @icon          HTML
 * @tags          html,encode,web
 */

const { encode } = require('@boop/he')

function main(input) {
	input.text = encode(input.text)
}
