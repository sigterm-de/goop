/**!
 * @name          URL Decode
 * @description   Decodes URL entities in your text.
 * @icon          link
 * @tags          url,decode,convert
 */

function main(input) {
	
	input.text = decodeURIComponent(input.text)
	
}