/**!
 * @name          Minify CSS
 * @description   Cleans and minifies CSS stylesheets.
 * @icon          broom
 * @tags          css,minify,clean,indent
 * @bias          -0.1
 */

const { cssmin } = require('@boop/vkBeautify')


function main(state) {
	state.text = cssmin(state.text)	
}
