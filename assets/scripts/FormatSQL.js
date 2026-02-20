/**!
 * @name          Format SQL
 * @description   Cleans and format SQL queries.
 * @icon          broom
 * @tags          mysql,sql,prettify,clean,indent
 * @bias          -0.1
 */

const { sql } = require('@boop/vkBeautify')


function main(state) {
	state.text = sql(state.text)	
}
