/**!
 * @name          Start Case
 * @description   Converts Your Text To Start Case.
 * @icon          type
 * @tags          start,case,function,lodash
 */

const { startCase } = require('@boop/lodash.boop')

function main(input) {
	
    input.text = startCase(input.text)
	
}
