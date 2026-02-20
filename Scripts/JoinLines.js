/**!
 * @name          Join Lines
 * @description   Joins all lines without any delimiter.
 * @icon          collapse
 * @tags          join
 */

function main(input) {
	input.text = input.text.replace(/\n/g, '');
}
