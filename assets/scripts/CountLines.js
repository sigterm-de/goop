/**!
 * @name          Count Lines
 * @description   Get the line count of your text
 * @icon          counter
 * @tags          count,length,size,line
 */

function main(input) {
	
	input.postInfo(`${input.text.split('\n').length} lines`)
	
}
