/**!
 * @name          URL Encode
 * @description   Encodes URL entities in your text.
 * @icon          link
 * @tags          url,encode,convert
 */

function main(input) {
	
	input.text = encodeURIComponent(input.text)
	
}