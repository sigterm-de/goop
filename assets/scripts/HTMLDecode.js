/**!
 * @name          HTML Decode
 * @description   Decodes HTML entities in your text
 * @icon          HTML
 * @tags          html,decode,web
 */

const { decode } = require('@boop/he')

function main(input) {
	input.text = decode(input.text)
}
