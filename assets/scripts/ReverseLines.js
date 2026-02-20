/**!
 * @name          Reverse Lines
 * @description   Flips every line of your text.
 * @icon          flip
 * @tags          reverse,order,invert,mirror,flip,upside,down
 */

function main(input) {
	input.text = input.text.split('\n').reverse().join('\n')
	
}
