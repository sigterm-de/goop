/**!
 * @name          Join Lines With Space
 * @description   Joins all lines with a space
 * @icon          collapse
 * @tags          join, space
 * @bias          -0.1
 */

function main(input) {
	input.text = input.text.replace(/\n/g, ' ');
}
